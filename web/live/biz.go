package live

import (
	"context"
	"fmt"
	"math"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/banbox/banbot/biz"
	"github.com/banbox/banbot/btime"
	"github.com/banbox/banbot/config"
	"github.com/banbox/banbot/core"
	"github.com/banbox/banbot/exg"
	"github.com/banbox/banbot/orm"
	"github.com/banbox/banbot/strat"
	"github.com/banbox/banbot/utils"
	"github.com/banbox/banbot/web/base"
	"github.com/banbox/banexg"
	utils2 "github.com/banbox/banexg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func regApiBiz(api fiber.Router) {
	api.Get("/version", getVersion)
	api.Get("/balance", getBalance)
	api.Get("/today_num", getTodayNum)
	api.Get("/statistics", getStatistics)
	api.Get("/incomes", getIncomes)
	api.Get("/task_pairs", getTaskPairs)
	api.Get("/orders", getOrders)
	api.Post("/forceexit", postForceExit)
	api.Post("/close_pos", postClosePos)
	api.Post("/delay_entry", postDelayEntry)
	api.Get("/config", getConfig)
	api.Post("/config", postConfig)
	api.Get("/stg_jobs", getStratJobs)
	api.Get("/performance", getPerformance)
	api.Get("/log", getLog)
	api.Get("/bot_info", getBotInfo)
}

type FnAccCB = func(acc string) error
type FnAccDbCB = func(acc string, sess *orm.Queries) error

func wrapAccount(c *fiber.Ctx, cb FnAccCB) error {
	account := c.Get("X-Account")
	if account == "" {
		return fiber.NewError(fiber.StatusBadRequest, "header `X-Account` missing")
	}
	return cb(account)
}

func wrapAccDb(c *fiber.Ctx, cb FnAccDbCB) error {
	return wrapAccount(c, func(acc string) error {
		ctx := context.Background()
		sess, conn, err := orm.Conn(ctx)
		if err != nil {
			return err
		}
		err_ := cb(acc, sess)
		conn.Release()
		return err_
	})
}

func getVersion(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version": core.Version,
	})
}

func getBalance(c *fiber.Ctx) error {
	return wrapAccount(c, func(account string) error {
		wallet := biz.GetWallets(account)
		items := make([]map[string]interface{}, 0)
		for coin, item := range wallet.Items {
			total := item.Total(true)
			items = append(items, map[string]interface{}{
				"symbol":     coin,
				"total":      total,
				"upol":       item.UnrealizedPOL,
				"free":       item.Available,
				"used":       total - item.Available,
				"total_fiat": item.FiatValue(true),
			})
		}
		return c.JSON(fiber.Map{
			"items": items,
			"total": wallet.FiatValue(true),
		})
	})
}

func getTodayNum(c *fiber.Ctx) error {
	return wrapAccDb(c, func(acc string, sess *orm.Queries) error {
		openOds, lock := orm.GetOpenODs(acc)
		lock.Lock()
		dayOpenNum := len(openOds)
		dayOpenPft := float64(0)
		for _, od := range openOds {
			dayOpenPft += od.Profit
		}
		lock.Unlock()

		// 获取今日完成的订单
		tfMSecs := int64(utils2.TFToSecs("1d") * 1000)
		nowMS := btime.UTCStamp()
		todayStartMS := utils2.AlignTfMSecs(nowMS, tfMSecs)
		taskId := orm.GetTaskID(acc)
		dayDoneNum := 0
		dayDonePft := float64(0)
		if taskId > 0 {
			orders, err := sess.GetOrders(orm.GetOrdersArgs{
				TaskID:      taskId,
				Status:      2, // 已完成状态
				CloseAfter:  todayStartMS,
				CloseBefore: nowMS,
			})
			if err != nil {
				return err
			}
			for _, od := range orders {
				dayDonePft += od.Profit
			}
			dayDoneNum = len(orders)
		}
		return c.JSON(fiber.Map{
			"running":    taskId > 0,
			"dayDoneNum": dayDoneNum,
			"dayDonePft": dayDonePft,
			"dayOpenNum": dayOpenNum,
			"dayOpenPft": dayOpenPft,
		})
	})
}

func getStatistics(c *fiber.Ctx) error {
	return wrapAccDb(c, func(acc string, sess *orm.Queries) error {
		taskId := orm.GetTaskID(acc)
		orders, err := sess.GetOrders(orm.GetOrdersArgs{
			TaskID: taskId,
		})
		if err != nil {
			return err
		}
		wallets := biz.GetWallets(acc)
		var totalDuration int64 // All order holding seconds 所有订单持仓秒数
		var profitSum, profitRateSum, totalCost float64
		var doneProfitSum, doneProfitRateSum, doneTotalCost float64
		var curMS = btime.UTCStamp()
		var odNum, winNum, lossNum, doneNum int
		var winValue, lossValue float64
		var bestPair string
		var bestRate float64
		var curDay int64
		var dayProfitSum float64
		var dayProfits []float64 // Daily Profit 每日利润
		var dayMSecs = int64(utils2.TFToSecs("1d") * 1000)
		for _, od := range orders {
			if od.Status < orm.InOutStatusPartEnter || od.Status > orm.InOutStatusFullExit {
				continue
			}
			odNum += 1
			durat := od.ExitAt - od.EnterAt
			if durat < 0 {
				durat = curMS - od.EnterAt
			}
			totalDuration += durat / 1000
			profitSum += od.Profit
			profitRateSum += od.ProfitRate
			totalCost += od.EnterCost()
			if od.Status == orm.InOutStatusFullExit {
				doneNum += 1
				if od.ProfitRate > bestRate {
					bestRate = od.ProfitRate
					bestPair = od.Symbol
				}
				if od.Profit > 0 {
					winNum += 1
					winValue += od.Profit
				} else {
					lossNum += 1
					lossValue -= od.Profit
				}
				doneProfitSum += od.Profit
				doneProfitRateSum += od.ProfitRate
				doneTotalCost += od.EnterCost()
				curDayMS := utils2.AlignTfMSecs(od.EnterAt, dayMSecs)
				if curDay == 0 || curDay == curDayMS {
					dayProfitSum += od.Profit
				} else {
					dayProfits = append(dayProfits, dayProfitSum)
					curDay = curDayMS
					dayProfitSum = 0
				}
			}
		}
		if dayProfitSum > 0 {
			dayProfits = append(dayProfits, dayProfitSum)
		}
		doneProfitMean := doneProfitSum / float64(max(1, doneNum))
		profitMean := profitSum / float64(max(1, odNum))
		profitFactor := winValue / max(1e-6, math.Abs(lossValue))
		winRate := float64(winNum) / float64(max(1, doneNum))

		firstEntMs, lastEntMs := int64(0), int64(0)
		if odNum > 0 {
			firstEntMs = orders[0].EnterAt
			lastEntMs = orders[len(orders)-1].EnterAt
		}

		expProfit, expRatio := utils.CalcExpectancy(dayProfits)
		initBalance := wallets.TotalLegal(nil, true) - profitSum
		ddPct, ddVal, _, _, _, _ := utils.CalcMaxDrawDown(dayProfits, initBalance)
		return c.JSON(fiber.Map{
			"doneProfitMean":    doneProfitMean,
			"doneProfitPctMean": doneProfitRateSum / float64(max(1, doneNum)) * 100,
			"doneProfitSum":     doneProfitSum,
			"doneProfitPctSum":  doneProfitSum / max(1e-6, doneTotalCost) * 100,
			"allProfitMean":     profitMean,
			"allProfitPctMean":  profitRateSum / float64(max(1, odNum)) * 100,
			"allProfitSum":      profitSum,
			"allProfitPctSum":   profitSum / max(1e-6, totalCost) * 100,
			"orderNum":          odNum,
			"doneOrderNum":      doneNum,
			"firstOdTs":         firstEntMs / 1000,
			"lastOdTs":          lastEntMs / 1000,
			"avgDuration":       totalDuration / int64(max(1, odNum)),
			"bestPair":          bestPair,
			"bestProfitPct":     bestRate,
			"winNum":            winNum,
			"lossNum":           lossNum,
			"profitFactor":      profitFactor,
			"winRate":           winRate,
			"expectancy":        expProfit,
			"expectancyRatio":   expRatio,
			"maxDrawdownPct":    ddPct * 100,
			"maxDrawdownVal":    ddVal,
			"totalCost":         totalCost,
			"botStartMs":        core.StartAt,
			"runTfs":            utils.KeysOfMap(core.TFSecs),
			"exchange":          core.ExgName,
			"market":            core.Market,
			"pairs":             core.Pairs,
		})
	})
}

func getOrders(c *fiber.Ctx) error {
	type OrderArgs struct {
		StartMs   int64  `query:"startMs"`
		StopMs    int64  `query:"stopMs"`
		Limit     int    `query:"limit"`
		AfterID   int    `query:"afterId"`
		Symbols   string `query:"symbols"`
		Status    string `query:"status"`
		Strategy  string `query:"strategy"`
		TimeFrame string `query:"timeFrame"`
		Source    string `query:"source" validate:"required"`
	}
	var data = new(OrderArgs)
	if err := base.VerifyArg(c, data, base.ArgQuery); err != nil {
		return err
	}
	type OdWrap struct {
		*orm.InOutOrder
		CurPrice float64 `json:"curPrice"`
	}
	getBotOrders := func(acc string, sess *orm.Queries) error {
		taskId := orm.GetTaskID(acc)
		var symbols []string
		if data.Symbols != "" {
			symbols = strings.Split(data.Symbols, ",")
		}
		var status = 0
		if data.Status == "open" {
			status = 1
		} else if data.Status == "his" {
			status = 2
		}
		orders, err := sess.GetOrders(orm.GetOrdersArgs{
			TaskID:      taskId,
			Strategy:    data.Strategy,
			Pairs:       symbols,
			TimeFrame:   data.TimeFrame,
			Status:      status,
			CloseAfter:  data.StartMs,
			CloseBefore: data.StopMs,
			Limit:       data.Limit,
			AfterID:     data.AfterID,
		})
		if err != nil {
			return err
		}
		odList := make([]*OdWrap, 0, len(orders))
		for _, od := range orders {
			price := float64(0)
			if od.ExitTag != "" && od.Exit != nil && od.Exit.Price > 0 {
				price = od.Exit.Price
			} else {
				price = core.GetPriceSafe(od.Symbol)
				if price > 0 {
					od.UpdateProfits(price)
				}
			}
			od.NanInfTo(0)
			odList = append(odList, &OdWrap{
				InOutOrder: od,
				CurPrice:   price,
			})
		}
		sort.Slice(odList, func(i, j int) bool {
			return odList[i].EnterAt > odList[j].EnterAt
		})
		return c.JSON(fiber.Map{
			"data": odList,
		})
	}
	getExgOrders := func(acc string, sess *orm.Queries) error {
		orders, err := exg.Default.FetchOrders(data.Symbols, data.StartMs, data.Limit, nil)
		if err != nil {
			return err
		}
		sort.Slice(orders, func(i, j int) bool {
			return orders[i].Timestamp > orders[j].Timestamp
		})
		return c.JSON(fiber.Map{
			"data": orders,
		})
	}
	getExgPositions := func(acc string, sess *orm.Queries) error {
		var symbols []string
		if data.Symbols != "" {
			symbols = strings.Split(data.Symbols, ",")
		}
		posList, err := exg.Default.FetchPositions(symbols, nil)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{
			"data": posList,
		})
	}
	return wrapAccDb(c, func(acc string, sess *orm.Queries) error {
		if data.Source == "bot" {
			return getBotOrders(acc, sess)
		} else if data.Source == "exchange" {
			return getExgOrders(acc, sess)
		} else if data.Source == "position" {
			return getExgPositions(acc, sess)
		} else {
			return fiber.NewError(fiber.StatusBadRequest, "invalid source")
		}
	})
}

func postForceExit(c *fiber.Ctx) error {
	type ForceExitArgs struct {
		OrderID string `json:"orderId" validate:"required"`
	}
	var data = new(ForceExitArgs)
	if err := base.VerifyArg(c, data, base.ArgBody); err != nil {
		return err
	}

	return wrapAccDb(c, func(acc string, sess *orm.Queries) error {
		openOds, lock := orm.GetOpenODs(acc)
		lock.Lock()

		var targetOrders []*orm.InOutOrder
		if data.OrderID == "all" {
			targetOrders = utils2.ValsOfMap(openOds)
		} else {
			orderID, err := strconv.ParseInt(data.OrderID, 10, 64)
			if err != nil {
				lock.Unlock()
				return fiber.NewError(fiber.StatusBadRequest, "invalid order id")
			}
			for _, od := range openOds {
				if od.ID == orderID {
					targetOrders = append(targetOrders, od)
					break
				}
			}
			if len(targetOrders) == 0 {
				lock.Unlock()
				return fiber.NewError(fiber.StatusNotFound, "order not found")
			}
		}
		lock.Unlock()

		odMgr := biz.GetLiveOdMgr(acc)
		closeNum, failNum := 0, 0
		var errMsg strings.Builder
		for _, od := range targetOrders {
			_, err2 := odMgr.ExitOrder(sess, od, &strat.ExitReq{
				Tag:      core.ExitTagUserExit,
				StgyName: od.Strategy,
				OrderID:  od.ID,
				Force:    true,
			})
			if err2 != nil {
				failNum += 1
				errMsg.WriteString(fmt.Sprintf("Order %v: %v\n", od.ID, err2.Short()))
			} else {
				closeNum += 1
			}
		}

		return c.JSON(fiber.Map{
			"closeNum": closeNum,
			"failNum":  failNum,
			"errMsg":   errMsg.String(),
		})
	})
}

func postClosePos(c *fiber.Ctx) error {
	type CloseArgs struct {
		Symbol    string  `json:"symbol" validate:"required"`
		Side      string  `json:"side"`
		Amount    float64 `json:"amount"`
		OrderType string  `json:"orderType"`
		Price     float64 `json:"price"`
	}
	var data = new(CloseArgs)
	if err := base.VerifyArg(c, data, base.ArgBody); err != nil {
		return err
	}
	var reqs []*CloseArgs
	if data.Symbol == "all" {
		posList, err := exg.Default.FetchPositions(nil, nil)
		if err != nil {
			return err
		}
		for _, p := range posList {
			reqs = append(reqs, &CloseArgs{
				Symbol:    p.Symbol,
				Side:      p.Side,
				Amount:    p.Contracts,
				OrderType: banexg.OdTypeMarket,
			})
		}
	} else {
		reqs = append(reqs, data)
	}
	closeNum, doneNum := 0, 0
	for _, q := range reqs {
		side := "sell"
		if q.Side == "short" {
			side = "buy"
		}
		params := map[string]interface{}{}
		if banexg.IsContract(core.Market) {
			posSide := "LONG"
			if q.Side == "short" {
				posSide = "SHORT"
			}
			params["positionSide"] = posSide
		}
		res, err := exg.Default.CreateOrder(q.Symbol, q.OrderType, side, q.Amount, q.Price, params)
		if err != nil {
			return err
		}
		if res.ID != "" {
			closeNum += 1
			if res.Filled == res.Amount {
				doneNum += 1
			}
		}
	}
	return c.JSON(fiber.Map{
		"closeNum": closeNum,
		"doneNum":  doneNum,
	})
}

func getIncomes(c *fiber.Ctx) error {
	type CloseArgs struct {
		InType    string `query:"intype" validate:"required"`
		Symbol    string `query:"symbol"`
		StartTime int64  `query:"startTime"`
		Limit     int    `query:"limit"`
	}
	var data = new(CloseArgs)
	if err := base.VerifyArg(c, data, base.ArgQuery); err != nil {
		return err
	}
	return wrapAccount(c, func(acc string) error {
		items, err := exg.Default.FetchIncomeHistory(data.InType, data.Symbol, data.StartTime, data.Limit, nil)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"data": items})
	})
}

func postDelayEntry(c *fiber.Ctx) error {
	type DelayArgs struct {
		Secs int64 `json:"secs" validate:"required"`
	}
	var data = new(DelayArgs)
	if err := base.VerifyArg(c, data, base.ArgBody); err != nil {
		return err
	}
	return wrapAccount(c, func(acc string) error {
		untilMS := btime.UTCStamp() + data.Secs*1000
		core.NoEnterUntil[acc] = untilMS
		return c.JSON(fiber.Map{
			"allowTradeAt": untilMS,
		})
	})
}

func getConfig(c *fiber.Ctx) error {
	data, err := config.DumpYaml()
	if err != nil {
		return err
	}
	return c.SendString(string(data))
}

func postConfig(c *fiber.Ctx) error {
	type PostArgs struct {
		Data string `json:"data" validate:"required"`
	}
	var data = new(PostArgs)
	if err_ := base.VerifyArg(c, data, base.ArgBody); err_ != nil {
		return err_
	}
	tempFile, err_ := os.CreateTemp("", "ban_web-*.yml")
	if err_ != nil {
		return err_
	}
	defer os.Remove(tempFile.Name())
	if _, err_ = tempFile.Write([]byte("Hello, world!")); err_ != nil {
		return err_
	}
	if err_ = tempFile.Close(); err_ != nil {
		return err_
	}
	args := config.Args
	args.Configs = []string{tempFile.Name()}
	err := config.LoadConfig(args)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"status": 200})
}

func getStratJobs(c *fiber.Ctx) error {
	type JobItem struct {
		Pair     string  `json:"pair"`
		Strategy string  `json:"strategy"`
		TF       string  `json:"tf"`
		Price    float64 `json:"price"`
		OdNum    int     `json:"odNum"`
	}
	return wrapAccount(c, func(acc string) error {
		jobs := strat.GetJobs(acc)
		items := make([]*JobItem, 0, len(jobs))
		openOds, lock := orm.GetOpenODs(acc)
		lock.Lock()
		defer lock.Unlock()
		for pairTF, jobMap := range jobs {
			arr := strings.Split(pairTF, "_")
			price := core.GetPriceSafe(arr[0])
			for stgName := range jobMap {
				var odNum = 0
				for _, od := range openOds {
					if od.Symbol == arr[0] && od.Timeframe == arr[1] && od.Strategy == stgName {
						odNum += 1
					}
				}
				item := &JobItem{
					Pair:     arr[0],
					TF:       arr[1],
					Strategy: stgName,
					Price:    price,
					OdNum:    odNum,
				}
				items = append(items, item)
			}
		}
		return c.JSON(fiber.Map{
			"jobs":   items,
			"strats": strat.Versions,
		})
	})
}

func getTaskPairs(c *fiber.Ctx) error {
	type PairArgs struct {
		Start int64 `query:"start"`
		Stop  int64 `query:"stop"`
	}
	var data = new(PairArgs)
	if err_ := base.VerifyArg(c, data, base.ArgQuery); err_ != nil {
		return err_
	}
	return wrapAccDb(c, func(acc string, sess *orm.Queries) error {
		ctx := context.Background()
		taskId := orm.GetTaskID(acc)
		if data.Stop == 0 {
			data.Stop = math.MaxInt64
		}
		pairs, err := sess.GetTaskPairs(ctx, orm.GetTaskPairsParams{
			TaskID:    int32(taskId),
			EnterAt:   data.Start,
			EnterAt_2: data.Stop,
		})
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"pairs": pairs})
	})
}

type GroupItem struct {
	Key       string  `json:"key"`
	HoldHours float64 `json:"holdHours"`
	TotalCost float64 `json:"totalCost"`
	ProfitSum float64 `json:"profitSum"`
	ProfitPct float64 `json:"profitPct"`
	CloseNum  int     `json:"closeNum"`
	WinNum    int     `json:"winNum"`
}

func getPerformance(c *fiber.Ctx) error {
	type PerfArgs struct {
		GroupBy   string   `query:"groupBy"`
		Pairs     []string `query:"pairs"`
		StartSecs int64    `query:"startSecs"`
		StopSecs  int64    `query:"stopSecs"`
		Limit     int      `query:"limit"`
	}
	var data = new(PerfArgs)
	if err_ := base.VerifyArg(c, data, base.ArgQuery); err_ != nil {
		return err_
	}
	return wrapAccDb(c, func(acc string, sess *orm.Queries) error {
		taskId := orm.GetTaskID(acc)
		orders, err := sess.GetOrders(orm.GetOrdersArgs{
			TaskID:      taskId,
			Pairs:       data.Pairs,
			Status:      2,
			CloseAfter:  data.StartSecs * 1000,
			CloseBefore: data.StopSecs * 1000,
		})
		if err != nil {
			return err
		}
		var odKey func(od *orm.InOutOrder) string
		if data.GroupBy == "symbol" {
			odKey = func(od *orm.InOutOrder) string {
				return od.Symbol
			}
		} else if data.GroupBy == "month" {
			tfMSecs := int64(utils2.TFToSecs("1M") * 1000)
			odKey = func(od *orm.InOutOrder) string {
				dateMS := utils2.AlignTfMSecs(od.EnterAt, tfMSecs)
				return btime.ToDateStr(dateMS, "2006-01")
			}
		} else if data.GroupBy == "week" {
			tfMSecs := int64(utils2.TFToSecs("1w") * 1000)
			odKey = func(od *orm.InOutOrder) string {
				dateMS := utils2.AlignTfMSecs(od.EnterAt, tfMSecs)
				return btime.ToDateStr(dateMS, "2006-01-02")
			}
		} else if data.GroupBy == "day" {
			tfMSecs := int64(utils2.TFToSecs("1d") * 1000)
			odKey = func(od *orm.InOutOrder) string {
				dateMS := utils2.AlignTfMSecs(od.EnterAt, tfMSecs)
				return btime.ToDateStr(dateMS, "2006-01-02")
			}
		} else {
			return c.JSON(fiber.Map{"code": 400, "msg": "unsupport group type: " + data.GroupBy})
		}
		res := groupOrders(orders, odKey)
		enterTags := groupOrders(orders, func(od *orm.InOutOrder) string {
			return od.EnterTag
		})
		exitTags := groupOrders(orders, func(od *orm.InOutOrder) string {
			return od.ExitTag
		})
		return c.JSON(fiber.Map{"items": res, "enters": enterTags, "exits": exitTags})
	})
}

func groupOrders(orders []*orm.InOutOrder, odKey func(od *orm.InOutOrder) string) []*GroupItem {
	var itemMap = map[string]*GroupItem{}
	hourMSecs := float64(utils2.TFToSecs("1h") * 1000)
	for _, od := range orders {
		key := odKey(od)
		gp, ok := itemMap[key]
		if !ok {
			gp = &GroupItem{Key: key}
			itemMap[key] = gp
		}
		holdHours := float64(od.ExitAt-od.EnterAt) / hourMSecs
		gp.CloseNum += 1
		gp.ProfitSum += od.Profit
		gp.TotalCost += od.EnterCost()
		gp.HoldHours += holdHours
		if od.Profit > 0 {
			gp.WinNum += 1
		}
	}
	for _, gp := range itemMap {
		if gp.TotalCost > 0 {
			gp.ProfitPct = gp.ProfitSum / gp.TotalCost
		}
		gp.HoldHours /= float64(gp.CloseNum)
	}
	var res = make([]*GroupItem, 0, len(itemMap))
	for _, v := range itemMap {
		res = append(res, v)
	}
	slices.SortFunc(res, func(a, b *GroupItem) int {
		if a.Key <= b.Key {
			return -1
		}
		return 1
	})
	return res
}

func getLog(c *fiber.Ctx) error {
	type LogArgs struct {
		Num int `query:"num"`
	}
	var data = new(LogArgs)
	if err_ := base.VerifyArg(c, data, base.ArgQuery); err_ != nil {
		return err_
	}
	if config.Args.Logfile == "" {
		return c.JSON(fiber.Map{"code": 400, "msg": "no log file"})
	}
	if data.Num == 0 {
		data.Num = 3000
	}
	lines, err := utils.ReadLastNLines(config.Args.Logfile, data.Num)
	if err != nil {
		return err
	}
	return c.SendString(strings.Join(lines, "\n"))
}

func getBotInfo(c *fiber.Ctx) error {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return err
	}
	v, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	return wrapAccount(c, func(acc string) error {
		stopUntil, _ := core.NoEnterUntil[acc]
		return c.JSON(fiber.Map{
			"cpuPct":       percent[0],
			"ramPct":       v.UsedPercent,
			"lastProcess":  core.LastBarMs,
			"allowTradeAt": stopUntil,
		})
	})
}
