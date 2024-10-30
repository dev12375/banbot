package data

import (
	"context"
	"fmt"
	"github.com/banbox/banbot/btime"
	"github.com/banbox/banbot/config"
	"github.com/banbox/banbot/core"
	"github.com/banbox/banbot/exg"
	"github.com/banbox/banbot/orm"
	"github.com/banbox/banbot/strat"
	"github.com/banbox/banbot/utils"
	"github.com/banbox/banexg"
	"github.com/banbox/banexg/errs"
	"github.com/banbox/banexg/log"
	utils2 "github.com/banbox/banexg/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"math"
	"slices"
	"strings"
)

type FnPairKline = func(bar *orm.InfoKline)
type FuncEnvEnd = func(bar *banexg.PairTFKline, adj *orm.AdjInfo)

type PairTFCache struct {
	TimeFrame  string
	TFSecs     int
	NextMS     int64         // Record the start timestamp of the next bar expected to be received. If it is inconsistent, the bar is missing and needs to be queried and updated. 记录下一个期待收到的bar起始时间戳，如果不一致，则出现了bar缺失，需查询更新。
	WaitBar    *banexg.Kline // Record unfinished bars. Should be set to nil when completed 记录尚未完成的bar。已完成时应置为nil
	Latest     *banexg.Kline // Record the latest bar data, which may not be completed or may be completed 记录最新bar数据，可能未完成，可能已完成
	AlignOffMS int64
}

/*
Feeder
Each Feeder corresponds to a trading pair. Can contain multiple time dimensions.

Supports dynamic addition of time dimension.
Backtest mode: Call execution callbacks in sequence according to the next update time of the Feeder.
Real mode: Subscribe to new data for this trading pair's time period and execute a callback when it is awakened.
Support warm-up data. Each strategy + trading pair is preheated independently throughout the entire process, and cross-preheating is not allowed to avoid btime contamination.
LiveFeeder requires preheating for both new trading pairs and new cycles; HistFeeder only requires preheating for new cycles.
每个Feeder对应一个交易对。可包含多个时间维度。

	支持动态添加时间维度。
	回测模式：根据Feeder的下次更新时间，按顺序调用执行回调。
	实盘模式：订阅此交易对时间周期的新数据，被唤起时执行回调。
	支持预热数据。每个策略+交易对全程单独预热，不可交叉预热，避免btime被污染。
	LiveFeeder新交易对和新周期都需要预热；HistFeeder仅新周期需要预热
*/
type Feeder struct {
	*orm.ExSymbol
	States   []*PairTFCache
	WaitBar  *banexg.Kline
	CallBack FnPairKline
	OnEnvEnd FuncEnvEnd                 // If the futures main force switches or the stock is ex-rights, the position needs to be closed first 期货主力切换或股票除权，需先平仓
	tfBars   map[string][]*banexg.Kline // Cache the original K-line of each cycle (not restored) 缓存各周期的原始K线（未复权）
	adjs     []*orm.AdjInfo             // List of weighting factors 复权因子列表
	adj      *orm.AdjInfo
	isWarmUp bool // Is it currently in preheating state? 当前是否预热状态
}

func (f *Feeder) getStates() []*PairTFCache {
	return f.States
}

func (f *Feeder) getSymbol() string {
	return f.Symbol
}

func (f *Feeder) getWaitBar() *banexg.Kline {
	return f.WaitBar
}

func (f *Feeder) setWaitBar(bar *banexg.Kline) {
	f.WaitBar = bar
}

/*
subTfs
Add monitoring to States and return the newly added TimeFrames
添加监听到States中，返回新增的TimeFrames
*/
func (f *Feeder) SubTfs(timeFrames []string, delOther bool) []string {
	var oldTfs = make(map[string]bool)
	var stateMap = make(map[string]*PairTFCache)
	var minTfSecs = 0 // 记录最小时间周期
	if len(f.States) > 0 {
		for _, sta := range f.States {
			oldTfs[sta.TimeFrame] = true
			stateMap[sta.TimeFrame] = sta
			if minTfSecs == 0 || sta.TFSecs < minTfSecs {
				minTfSecs = sta.TFSecs
			}
		}
	}
	// New records are added to adds, existing ones are deleted from oldTfs, and stateMap retains all
	// 新增的记录到adds中，已有的从oldTfs中删除，stateMap保留全部的
	exchange, err := exg.GetWith(f.Exchange, f.Market, "")
	if err != nil {
		log.Warn("get exchange fail", zap.String("ex", f.Exchange), zap.Error(err))
		return nil
	}
	exgID := exchange.Info().ID
	adds := make([]string, 0, len(timeFrames))
	for _, tf := range timeFrames {
		if _, ok := oldTfs[tf]; ok {
			delete(oldTfs, tf)
			continue
		}
		tfSecs := utils2.TFToSecs(tf)
		sta := &PairTFCache{
			TimeFrame:  tf,
			TFSecs:     tfSecs,
			AlignOffMS: int64(exg.GetAlignOff(exgID, tfSecs) * 1000),
		}
		stateMap[tf] = sta
		if minTfSecs == 0 || sta.TFSecs < minTfSecs {
			minTfSecs = sta.TFSecs
		}
		adds = append(adds, tf)
	}
	// If you need to delete the unintroduced state, record the state of the minimum period to prevent rebuilding from blank again.
	// 如果需要删除未传入的，记录下最小周期的state，防止再次从空白重建
	var minDel *PairTFCache
	if delOther && len(oldTfs) > 0 {
		// Delete the time period passed in this time
		// 删除此次为传入的时间周期
		for tf := range oldTfs {
			if sta, ok := stateMap[tf]; ok {
				if sta.TFSecs == minTfSecs {
					minDel = sta
				}
				delete(stateMap, tf)
			}
		}
	}
	var newStates = utils.ValsOfMap(stateMap)
	// Sort all periods from small to large. The first one must be the least common multiple of all subsequent states, so that all subsequent states can be updated from the first one.
	// 对所有周期从小到大排序，第一个必须是后续所有states的最小公倍数，以便能从第一个更新后续所有
	slices.SortFunc(newStates, func(a, b *PairTFCache) int {
		return a.TFSecs - b.TFSecs
	})
	secs := make([]int, len(newStates))
	for i, v := range newStates {
		secs[i] = v.TFSecs
	}
	minSecs := utils.GcdInts(secs)
	if minSecs != newStates[0].TFSecs {
		if minDel != nil && minDel.TFSecs == minSecs {
			newStates = append([]*PairTFCache{minDel}, newStates...)
		} else {
			minTf := utils2.SecsToTF(minSecs)
			newStates = append([]*PairTFCache{{TFSecs: minSecs, TimeFrame: minTf}}, newStates...)
		}
	}
	f.States = newStates
	return adds
}

/*
Update State and trigger callback (internal automatic restoration)
bars original unweighted K-line
更新State并触发回调（内部自动复权）
bars 原始未复权的K线
*/
func (f *Feeder) onStateOhlcvs(state *PairTFCache, bars []*banexg.Kline, lastOk bool) []*banexg.Kline {
	if len(bars) == 0 {
		return nil
	}
	finishBars := bars
	if !lastOk {
		finishBars = bars[:len(bars)-1]
	}
	if state.WaitBar != nil && state.WaitBar.Time < bars[0].Time {
		finishBars = append([]*banexg.Kline{state.WaitBar}, finishBars...)
	}
	state.Latest = bars[len(bars)-1]
	state.WaitBar = nil
	if !lastOk {
		state.WaitBar = state.Latest
	}
	tfMSecs := int64(state.TFSecs * 1000)
	if len(finishBars) > 0 {
		state.NextMS = finishBars[len(finishBars)-1].Time + tfMSecs
		f.addTfKlines(state.TimeFrame, finishBars)
		adjBars := f.adj.Apply(finishBars, core.AdjFront)
		f.fireCallBacks(state.TimeFrame, tfMSecs, adjBars, f.adj)
	}
	return finishBars
}

func (f *Feeder) getTfKlines(tf string, endMS int64, limit int, pBar *utils.PrgBar) ([]*banexg.Kline, *errs.Error) {
	bars, _ := f.tfBars[tf]
	if len(bars) > 0 {
		// 缓存有，直接返回
		bars = orm.ApplyAdj(f.adjs, bars, core.AdjFront, endMS, limit)
		if pBar != nil {
			pBar.Add(core.StepTotal)
		}
		return bars, nil
	}
	exchange, err := exg.GetWith(f.Exchange, f.Market, "")
	if err != nil {
		return nil, err
	}
	adjs, bars, err := orm.AutoFetchOHLCV(exchange, f.ExSymbol, tf, 0, endMS, limit, false, pBar)
	if err != nil {
		return nil, err
	}
	f.tfBars[tf] = bars
	bars = orm.ApplyAdj(adjs, bars, core.AdjFront, 0, 0)
	return bars, nil
}

func (f *Feeder) addTfKlines(tf string, bars []*banexg.Kline) {
	olds, _ := f.tfBars[tf]
	if len(olds) > core.NumTaCache*2 {
		olds = olds[len(olds)-core.NumTaCache*3/2:]
	}
	f.tfBars[tf] = append(olds, bars...)
}

func (f *Feeder) fireCallBacks(timeFrame string, tfMSecs int64, bars []*banexg.Kline, adj *orm.AdjInfo) {
	isLive := core.LiveMode
	pair := f.Symbol
	for _, bar := range bars {
		if !isLive {
			btime.CurTimeMS = bar.Time + tfMSecs
		}
		f.CallBack(&orm.InfoKline{
			PairTFKline: &banexg.PairTFKline{Kline: *bar, Symbol: pair, TimeFrame: timeFrame},
			Adj:         adj,
			IsWarmUp:    f.isWarmUp,
		})
	}
	if isLive && !f.isWarmUp {
		// 记录收到的bar数量
		hits, ok := core.TfPairHits[timeFrame]
		if !ok {
			hits = make(map[string]int)
			core.TfPairHits[timeFrame] = hits
		}
		num, _ := hits[pair]
		hits[pair] = num + len(bars)
		// 检查是否延迟
		lastTime := bars[len(bars)-1].Time
		delay := btime.TimeMS() - (lastTime + tfMSecs)
		if delay > tfMSecs && tfMSecs >= 60000 {
			barNum := delay / tfMSecs
			log.Warn(fmt.Sprintf("%s/%s bar too late, delay %v bars, %v", pair, timeFrame, barNum, lastTime))
		}
	}
}

type IKlineFeeder interface {
	getSymbol() string
	getWaitBar() *banexg.Kline
	setWaitBar(bar *banexg.Kline)
	/*
		SubTfs Subscribe to data for a specified time period for the current target. Multiple 为当前标的订阅指定时间周期的数据，可多个
	*/
	SubTfs(timeFrames []string, delOther bool) []string
	/*
		WarmTfs The preheating time period gives the number of K lines to the specified time. 预热时间周期给定K线数量到指定时间
	*/
	WarmTfs(curMS int64, tfNums map[string]int, pBar *utils.PrgBar) (int64, *errs.Error)
	onNewBars(barTfMSecs int64, bars []*banexg.Kline) (bool, *errs.Error)
	getStates() []*PairTFCache
}

/*
KlineFeeder
Each Feeder corresponds to a trading pair. Can contain multiple time dimensions. Real use.

Supports dynamic addition of time dimension.
Supports returning preheating data. Each strategy + trading pair is preheated independently throughout the entire process, and cross-preheating is not allowed to avoid btime contamination.

Backtest mode: Use derived structure: DbKlineFeeder

Real mode: Subscribe to new data for this trading pair's time period and execute a callback when it is awakened.
Check whether this trading pair has been refreshed in the spider monitor. If not, send a message to the crawler monitor.
每个Feeder对应一个交易对。可包含多个时间维度。实盘使用。

	支持动态添加时间维度。
	支持返回预热数据。每个策略+交易对全程单独预热，不可交叉预热，避免btime被污染。

	回测模式：使用派生结构体：DbKlineFeeder

	实盘模式：订阅此交易对时间周期的新数据，被唤起时执行回调。
	检查此交易对是否已在spider监听刷新，如没有则发消息给爬虫监听。
*/
type KlineFeeder struct {
	Feeder
	PreFire  float64        // Ratio of triggering bar early 提前触发bar的比率
	adjIdx   int            // adjs的索引
	warmNums map[string]int // Preheating quantity in each cycle 各周期预热数量
	showLog  bool
}

func NewKlineFeeder(exs *orm.ExSymbol, callBack FnPairKline, showLog bool) (*KlineFeeder, *errs.Error) {
	sess, conn, err := orm.Conn(nil)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	adjs, err := sess.GetAdjs(exs.ID)
	if err != nil {
		return nil, err
	}
	return &KlineFeeder{
		Feeder: Feeder{
			ExSymbol: exs,
			CallBack: callBack,
			tfBars:   make(map[string][]*banexg.Kline),
			adjs:     adjs,
		},
		PreFire: config.PreFire,
		showLog: showLog,
	}, nil
}

func (f *KlineFeeder) WarmTfs(curMS int64, tfNums map[string]int, pBar *utils.PrgBar) (int64, *errs.Error) {
	if len(tfNums) == 0 {
		tfNums = f.warmNums
		if len(tfNums) == 0 {
			return 0, nil
		}
	} else {
		f.warmNums = tfNums
	}
	maxEndMs := int64(0)
	for tf, warmNum := range tfNums {
		tfMSecs := int64(utils2.TFToSecs(tf) * 1000)
		if tfMSecs < int64(60000) || warmNum <= 0 {
			continue
		}
		endMS := utils2.AlignTfMSecs(curMS, tfMSecs)
		bars, err := f.getTfKlines(tf, endMS, warmNum, pBar)
		if err != nil {
			return 0, err
		}
		if len(bars) == 0 && f.showLog {
			log.Info("skip warm as empty", zap.String("pair", f.Symbol), zap.String("tf", tf),
				zap.Int("want", warmNum), zap.Int64("end", endMS))
			continue
		}
		if warmNum != len(bars) && f.showLog {
			barEndMs := bars[len(bars)-1].Time + tfMSecs
			barStartMs := bars[0].Time
			lackNum := warmNum - len(bars)
			log.Info(fmt.Sprintf("warm %s/%s lack %v bars, expect: %v, range:%v-%v", f.Symbol,
				tf, lackNum, warmNum, barStartMs, barEndMs))
		}
		curEnd := f.warmTf(tf, bars)
		maxEndMs = max(maxEndMs, curEnd)
	}
	return maxEndMs, nil
}

/*
WarmTfs
Warm-up cycle data. When dynamically adding cycles to an existing HistDataFeeder, this method should be called to warm up the data.
If TaEnv already exists it will be reset.

LiveFeeder should also call this function when initializing
The incoming bars are the K-lines after the restoration of rights.

Returns the ending timestamp (i.e. the starting timestamp of the next bar)
预热周期数据。当动态添加周期到已有的HistDataFeeder时，应调用此方法预热数据。
如果TaEnv已存在会被重置。

	LiveFeeder在初始化时也应当调用此函数
	传入的bars是复权后的K线

返回结束的时间戳（即下一个bar开始时间戳）
*/
func (f *KlineFeeder) warmTf(tf string, bars []*banexg.Kline) int64 {
	if len(bars) == 0 {
		return 0
	}
	f.isWarmUp = true
	tfMSecs := int64(utils2.TFToSecs(tf) * 1000)
	lastMS := bars[len(bars)-1].Time + tfMSecs
	envKey := strings.Join([]string{f.Symbol, tf}, "_")
	if env, ok := strat.Envs[envKey]; ok {
		env.Reset()
	}
	if len(f.adjs) > 0 {
		// 按复权信息分批调用
		cache := make([]*banexg.Kline, 0, len(bars))
		var pAdj = f.adjs[0]
		var pi = 1
		forEnd := false
		for i, k := range bars {
			for k.Time >= pAdj.StopMS {
				if len(cache) > 0 {
					f.fireCallBacks(tf, tfMSecs, cache, pAdj)
					cache = make([]*banexg.Kline, 0, len(bars))
				}
				if pi >= len(f.adjs) {
					f.fireCallBacks(tf, tfMSecs, bars[i:], nil)
					forEnd = true
					pAdj = nil
					break
				}
				pAdj = f.adjs[pi]
				pi += 1
			}
			if forEnd {
				break
			}
			cache = append(cache, k)
		}
		if len(cache) > 0 {
			f.fireCallBacks(tf, tfMSecs, cache, pAdj)
		}
	} else {
		f.fireCallBacks(tf, tfMSecs, bars, nil)
	}
	for _, sta := range f.States {
		if sta.TimeFrame == tf {
			sta.NextMS = lastMS
			break
		}
	}
	f.isWarmUp = false
	return lastMS
}

/*
onNewBars
There is newly completed sub-period candle data, try to update
bars are K lines that have not been re-righted and will be re-righted internally.
有新完成的子周期蜡烛数据，尝试更新
bars 是未复权的K线，内部会进行复权
*/
func (f *KlineFeeder) onNewBars(barTfMSecs int64, bars []*banexg.Kline) (bool, *errs.Error) {
	state := f.States[0]
	staMSecs := int64(state.TFSecs * 1000)
	var ohlcvs []*banexg.Kline
	var lastOk bool
	if barTfMSecs < staMSecs {
		var olds []*banexg.Kline
		if state.WaitBar != nil {
			olds = append(olds, state.WaitBar)
		}
		ohlcvs, lastOk = utils.BuildOHLCV(bars, staMSecs, f.PreFire, olds, barTfMSecs, state.AlignOffMS)
	} else if barTfMSecs == staMSecs {
		ohlcvs, lastOk = bars, true
	} else {
		barTf := utils2.SecsToTF(int(barTfMSecs / 1000))
		msg := fmt.Sprintf("bar intv invalid, expect %v, cur: %v", state.TimeFrame, barTf)
		return false, errs.NewMsg(core.ErrInvalidBars, msg)
	}
	if len(ohlcvs) == 0 {
		return false, nil
	}
	minState, minOhlcvs := state, ohlcvs
	// 应该按周期从大到小触发
	if len(f.States) > 1 {
		// For the 2nd and subsequent coarse grains. OHLC updates from the first
		// Even if the first one is not completed, the coarser period dimension must be updated, otherwise data loss will occur
		// 对于第2个及后续的粗粒度。从第一个得到的OHLC更新
		// 即使第一个没有完成，也要更新更粗周期维度，否则会造成数据丢失
		if barTfMSecs < staMSecs {
			// The last unfinished data should be kept here
			// 这里应该保留最后未完成的数据
			ohlcvs, _ = utils.BuildOHLCV(bars, staMSecs, f.PreFire, nil, barTfMSecs, state.AlignOffMS)
		} else {
			ohlcvs = bars
		}
		for i := len(f.States) - 1; i >= 1; i-- {
			state = f.States[i]
			var olds []*banexg.Kline
			if state.WaitBar != nil {
				olds = append(olds, state.WaitBar)
			}
			bigTfMSecs := int64(state.TFSecs * 1000)
			curOhlcvs, lastOk := utils.BuildOHLCV(ohlcvs, bigTfMSecs, f.PreFire, olds, staMSecs, state.AlignOffMS)
			f.onStateOhlcvs(state, curOhlcvs, lastOk)
		}
	}
	//Subsequence period dimension <= current dimension. When receiving the data sent by the spider, there may be 3 or more ohlcvs
	//子序列周期维度<=当前维度。当收到spider发送的数据时，这里可能是3个或更多ohlcvs
	doneBars := f.onStateOhlcvs(minState, minOhlcvs, lastOk)
	return len(doneBars) > 0, nil
}

type IHistKlineFeeder interface {
	IKlineFeeder
	getNextMS() int64
	/*
		DownIfNeed Download the entire range of K lines, which needs to be called before SetSeek  下载整个范围的K线，需在SetSeek前调用
	*/
	DownIfNeed(sess *orm.Queries, exchange banexg.BanExchange, pBar *utils.PrgBar) *errs.Error
	/*
		SetSeek Set the reading position and call it before loop reading   设置读取位置，在循环读取前调用
	*/
	SetSeek(since int64)
	/*
		GetBar Get the current K line, and then call CallNext to move the pointer to the next 获取当前K线，然后可调用CallNext移动指针到下一个
	*/
	GetBar() *banexg.Kline
	/*
		RunBar Run the callback function corresponding to Bar 运行Bar对应的回调函数
	*/
	RunBar(bar *banexg.Kline) *errs.Error
	/*
		CallNext Move the pointer to the next K line 移动指针到下一个K线
	*/
	CallNext()
}

/*
HistKLineFeeder
Historical data feedback device. Is the base class for file feedback and database feedback.

Backtest mode: Read 3K bars each time, and backtest triggers in sequence according to nextMS size.
历史数据反馈器。是文件反馈器和数据库反馈器的基类。

	回测模式：每次读取3K个bar，按nextMS大小依次回测触发。
*/
type HistKLineFeeder struct {
	KlineFeeder
	TimeRange  *config.TimeTuple
	rowIdx     int             // The index of the next Bar in the cache, -1 means it has ended 缓存中下一个Bar的索引，-1表示已结束
	caches     []*banexg.Kline // Cached Bar, fire one by one, reload after reading 缓存的Bar，逐个fire，读取完重新加载
	nextMS     int64           // The 13-digit millisecond timestamp of the next bar, math.MaxInt32 indicates the end 下一个bar的13位毫秒时间戳，math.MaxInt32表示结束
	minGapMs   int64           // Minimum number of milliseconds between caches caches中最小的间隔毫秒数
	setNext    func()
	TradeTimes [][2]int64 // Trading time 可交易时间
}

func (f *HistKLineFeeder) getNextMS() int64 {
	return f.nextMS
}

/*
Get the current bar for invokeBar; callNext should be called afterwards to set the cursor to the next bar.
获取当前bar，用于invokeBar；之后应调用callNext设置光标到下一个bar
*/
func (f *HistKLineFeeder) GetBar() *banexg.Kline {
	if f.rowIdx < 0 || f.rowIdx >= len(f.caches) {
		return nil
	}
	bar := f.caches[f.rowIdx]
	return bar
}

func (f *HistKLineFeeder) RunBar(bar *banexg.Kline) *errs.Error {
	_, err := f.onNewBars(f.minGapMs, []*banexg.Kline{bar})
	return err
}

func (f *HistKLineFeeder) CallNext() {
	f.setNext()
}

/*
DBKlineFeeder
The database reads the K-line Feeder for backtesting
数据库读取K线的Feeder，用于回测
*/
type DBKlineFeeder struct {
	HistKLineFeeder
	offsetMS int64
}

func NewDBKlineFeeder(exs *orm.ExSymbol, callBack FnPairKline, showLog bool) (*DBKlineFeeder, *errs.Error) {
	exchange, err := exg.GetWith(exs.Exchange, exs.Market, "")
	if err != nil {
		return nil, err
	}
	var tradeTimes [][2]int64
	market, err := exchange.GetMarket(exs.Symbol)
	if err == nil {
		tradeTimes = market.GetTradeTimes()
	}
	feeder, err := NewKlineFeeder(exs, callBack, showLog)
	if err != nil {
		return nil, err
	}
	res := &DBKlineFeeder{
		HistKLineFeeder: HistKLineFeeder{
			KlineFeeder: *feeder,
			TimeRange:   config.TimeRange,
			TradeTimes:  tradeTimes,
		},
	}
	res.setNext = makeSetNext(res)
	return res, nil
}

func (f *DBKlineFeeder) SetSeek(since int64) {
	if since == 0 {
		// 这里不能为0，不然会从后往前读取K线，导致缺失
		since = core.MSMinStamp
	}
	f.rowIdx = 0
	f.nextMS = 0
	f.offsetMS = since
	f.setNext()
}

/*
DownIfNeed
Download data for a specified interval
pBar is used for progress update, the total is 1000, and the amount is updated each time
下载指定区间的数据
pBar 用于进度更新，总和为1000，每次更新此次的量
*/
func (f *DBKlineFeeder) DownIfNeed(sess *orm.Queries, exchange banexg.BanExchange, pBar *utils.PrgBar) *errs.Error {
	if len(f.States) == 0 || f.DelistMs > 0 {
		return nil
	}
	downTf, err := orm.GetDownTF(f.States[0].TimeFrame)
	if err != nil {
		if pBar != nil {
			pBar.Add(core.StepTotal)
		}
		return err
	}
	if sess == nil {
		ctx := context.Background()
		var conn *pgxpool.Conn
		sess, conn, err = orm.Conn(ctx)
		if err != nil {
			if pBar != nil {
				pBar.Add(core.StepTotal)
			}
			return err
		}
		defer conn.Release()
	}
	_, err = sess.DownOHLCV2DB(exchange, f.ExSymbol, downTf, f.TimeRange.StartMS, f.TimeRange.EndMS, pBar)
	return err
}

func (f *DBKlineFeeder) setAdjIdx() {
	for f.adjIdx < len(f.adjs) {
		adj := f.adjs[f.adjIdx]
		if f.nextMS < adj.StopMS {
			f.adj = adj
			return
		}
		f.adjIdx += 1
	}
	f.adj = nil
}

func makeSetNext(f *DBKlineFeeder) func() {
	return func() {
		if f.rowIdx+1 < len(f.caches) {
			f.rowIdx += 1
			f.nextMS = f.caches[f.rowIdx].Time
			if f.adj != nil && f.nextMS >= f.adj.StopMS {
				old := f.caches[f.rowIdx-1]
				tf := f.States[0].TimeFrame
				f.OnEnvEnd(&banexg.PairTFKline{
					Symbol:    f.Symbol,
					TimeFrame: tf,
					Kline:     *old,
				}, f.adj)
				// Warm up again
				// 重新复权预热
				_, err := f.WarmTfs(f.nextMS, nil, nil)
				if err != nil {
					log.Error("next warm tf fail", zap.Error(err))
				}
				f.setAdjIdx()
			}
			return
		}
		// After the cache reading is completed, re-read the database
		// 缓存读取完毕，重新读取数据库
		state := f.States[0]
		tfMSecs := int64(state.TFSecs * 1000)
		sess, conn, err := orm.Conn(nil)
		if err != nil {
			f.rowIdx = -1
			f.nextMS = math.MaxInt64
			log.Error("get conn fail while loading kline", zap.Error(err))
			return
		}
		defer conn.Release()
		endMS := f.TimeRange.EndMS
		if endMS > 0 && f.nextMS+tfMSecs >= endMS {
			f.rowIdx = -1
			f.nextMS = math.MaxInt64
			return
		}
		batchSize := 3000
		_, bars, err := sess.GetOHLCV(f.ExSymbol, state.TimeFrame, f.offsetMS, endMS, batchSize, false)
		if err != nil || len(bars) == 0 {
			f.rowIdx = -1
			f.nextMS = math.MaxInt64
			if err != nil {
				log.Error("load kline fail", zap.Error(err))
			}
			return
		}
		f.caches = bars
		f.rowIdx = 0
		f.nextMS = bars[0].Time
		f.offsetMS = bars[len(bars)-1].Time + tfMSecs
		f.setAdjIdx()
		f.minGapMs = math.MaxInt64
		for i, b := range bars[1:] {
			gap := b.Time - bars[i].Time
			if gap < f.minGapMs {
				f.minGapMs = gap
			}
		}
	}
}
