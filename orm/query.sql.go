// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: query.sql

package orm

import (
	"context"
)

type AddAdjFactorsParams struct {
	Sid     int32   `json:"sid"`
	SubID   int32   `json:"sub_id"`
	StartMs int64   `json:"start_ms"`
	Factor  float64 `json:"factor"`
}

type AddCalendarsParams struct {
	Name    string `json:"name"`
	StartMs int64  `json:"start_ms"`
	StopMs  int64  `json:"stop_ms"`
}

const addExOrder = `-- name: AddExOrder :one
insert into exorder ("task_id", "inout_id", "symbol", "enter", "order_type", "order_id", "side",
                     "create_at", "price", "average", "amount", "filled", "status", "fee", "fee_type", "update_at")
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
    RETURNING id
`

type AddExOrderParams struct {
	TaskID    int32   `json:"task_id"`
	InoutID   int32   `json:"inout_id"`
	Symbol    string  `json:"symbol"`
	Enter     bool    `json:"enter"`
	OrderType string  `json:"order_type"`
	OrderID   string  `json:"order_id"`
	Side      string  `json:"side"`
	CreateAt  int64   `json:"create_at"`
	Price     float64 `json:"price"`
	Average   float64 `json:"average"`
	Amount    float64 `json:"amount"`
	Filled    float64 `json:"filled"`
	Status    int16   `json:"status"`
	Fee       float64 `json:"fee"`
	FeeType   string  `json:"fee_type"`
	UpdateAt  int64   `json:"update_at"`
}

func (q *Queries) AddExOrder(ctx context.Context, arg AddExOrderParams) (int64, error) {
	row := q.db.QueryRow(ctx, addExOrder,
		arg.TaskID,
		arg.InoutID,
		arg.Symbol,
		arg.Enter,
		arg.OrderType,
		arg.OrderID,
		arg.Side,
		arg.CreateAt,
		arg.Price,
		arg.Average,
		arg.Amount,
		arg.Filled,
		arg.Status,
		arg.Fee,
		arg.FeeType,
		arg.UpdateAt,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const addIOrder = `-- name: AddIOrder :one
insert into iorder ("task_id", "symbol", "sid", "timeframe", "short", "status",
                    "enter_tag", "init_price", "quote_cost", "exit_tag", "leverage",
                    "enter_at", "exit_at", "strategy", "stg_ver", "max_pft_rate", "max_draw_down",
                    "profit_rate", "profit", "info")
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
    RETURNING id
`

type AddIOrderParams struct {
	TaskID      int32   `json:"task_id"`
	Symbol      string  `json:"symbol"`
	Sid         int32   `json:"sid"`
	Timeframe   string  `json:"timeframe"`
	Short       bool    `json:"short"`
	Status      int16   `json:"status"`
	EnterTag    string  `json:"enter_tag"`
	InitPrice   float64 `json:"init_price"`
	QuoteCost   float64 `json:"quote_cost"`
	ExitTag     string  `json:"exit_tag"`
	Leverage    float64 `json:"leverage"`
	EnterAt     int64   `json:"enter_at"`
	ExitAt      int64   `json:"exit_at"`
	Strategy    string  `json:"strategy"`
	StgVer      int32   `json:"stg_ver"`
	MaxPftRate  float64 `json:"max_pft_rate"`
	MaxDrawDown float64 `json:"max_draw_down"`
	ProfitRate  float64 `json:"profit_rate"`
	Profit      float64 `json:"profit"`
	Info        string  `json:"info"`
}

func (q *Queries) AddIOrder(ctx context.Context, arg AddIOrderParams) (int64, error) {
	row := q.db.QueryRow(ctx, addIOrder,
		arg.TaskID,
		arg.Symbol,
		arg.Sid,
		arg.Timeframe,
		arg.Short,
		arg.Status,
		arg.EnterTag,
		arg.InitPrice,
		arg.QuoteCost,
		arg.ExitTag,
		arg.Leverage,
		arg.EnterAt,
		arg.ExitAt,
		arg.Strategy,
		arg.StgVer,
		arg.MaxPftRate,
		arg.MaxDrawDown,
		arg.ProfitRate,
		arg.Profit,
		arg.Info,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const addInsKline = `-- name: AddInsKline :one
insert into ins_kline ("sid", "timeframe", "start_ms", "stop_ms")
values ($1, $2, $3, $4) RETURNING id
`

type AddInsKlineParams struct {
	Sid       int32  `json:"sid"`
	Timeframe string `json:"timeframe"`
	StartMs   int64  `json:"start_ms"`
	StopMs    int64  `json:"stop_ms"`
}

func (q *Queries) AddInsKline(ctx context.Context, arg AddInsKlineParams) (int32, error) {
	row := q.db.QueryRow(ctx, addInsKline,
		arg.Sid,
		arg.Timeframe,
		arg.StartMs,
		arg.StopMs,
	)
	var id int32
	err := row.Scan(&id)
	return id, err
}

type AddKHolesParams struct {
	Sid       int32  `json:"sid"`
	Timeframe string `json:"timeframe"`
	Start     int64  `json:"start"`
	Stop      int64  `json:"stop"`
}

const addKInfo = `-- name: AddKInfo :one
insert into kinfo
("sid", "timeframe", "start", "stop")
values ($1, $2, $3, $4)
    returning sid, timeframe, start, stop
`

type AddKInfoParams struct {
	Sid       int32  `json:"sid"`
	Timeframe string `json:"timeframe"`
	Start     int64  `json:"start"`
	Stop      int64  `json:"stop"`
}

func (q *Queries) AddKInfo(ctx context.Context, arg AddKInfoParams) (*KInfo, error) {
	row := q.db.QueryRow(ctx, addKInfo,
		arg.Sid,
		arg.Timeframe,
		arg.Start,
		arg.Stop,
	)
	var i KInfo
	err := row.Scan(
		&i.Sid,
		&i.Timeframe,
		&i.Start,
		&i.Stop,
	)
	return &i, err
}

type AddSymbolsParams struct {
	Exchange string `json:"exchange"`
	ExgReal  string `json:"exg_real"`
	Market   string `json:"market"`
	Symbol   string `json:"symbol"`
}

const addTask = `-- name: AddTask :one
insert into bottask
("mode", "name", "create_at", "start_at", "stop_at", "info")
values ($1, $2, $3, $4, $5, $6)
returning id, mode, name, create_at, start_at, stop_at, info
`

type AddTaskParams struct {
	Mode     string `json:"mode"`
	Name     string `json:"name"`
	CreateAt int64  `json:"create_at"`
	StartAt  int64  `json:"start_at"`
	StopAt   int64  `json:"stop_at"`
	Info     string `json:"info"`
}

func (q *Queries) AddTask(ctx context.Context, arg AddTaskParams) (*BotTask, error) {
	row := q.db.QueryRow(ctx, addTask,
		arg.Mode,
		arg.Name,
		arg.CreateAt,
		arg.StartAt,
		arg.StopAt,
		arg.Info,
	)
	var i BotTask
	err := row.Scan(
		&i.ID,
		&i.Mode,
		&i.Name,
		&i.CreateAt,
		&i.StartAt,
		&i.StopAt,
		&i.Info,
	)
	return &i, err
}

const delAdjFactors = `-- name: DelAdjFactors :exec
delete from adj_factors
where sid=$1
`

func (q *Queries) DelAdjFactors(ctx context.Context, sid int32) error {
	_, err := q.db.Exec(ctx, delAdjFactors, sid)
	return err
}

const delInsKline = `-- name: DelInsKline :exec
delete from ins_kline
where id=$1
`

func (q *Queries) DelInsKline(ctx context.Context, id int32) error {
	_, err := q.db.Exec(ctx, delInsKline, id)
	return err
}

const delKHoleRange = `-- name: DelKHoleRange :exec
delete from khole
where sid = $1 and timeframe=$2 and start >= $3 and stop <= $4
`

type DelKHoleRangeParams struct {
	Sid       int32  `json:"sid"`
	Timeframe string `json:"timeframe"`
	Start     int64  `json:"start"`
	Stop      int64  `json:"stop"`
}

func (q *Queries) DelKHoleRange(ctx context.Context, arg DelKHoleRangeParams) error {
	_, err := q.db.Exec(ctx, delKHoleRange,
		arg.Sid,
		arg.Timeframe,
		arg.Start,
		arg.Stop,
	)
	return err
}

const findTask = `-- name: FindTask :one
select id, mode, name, create_at, start_at, stop_at, info from bottask
where mode = $1 and name = $2
order by create_at desc
limit 1
`

type FindTaskParams struct {
	Mode string `json:"mode"`
	Name string `json:"name"`
}

func (q *Queries) FindTask(ctx context.Context, arg FindTaskParams) (*BotTask, error) {
	row := q.db.QueryRow(ctx, findTask, arg.Mode, arg.Name)
	var i BotTask
	err := row.Scan(
		&i.ID,
		&i.Mode,
		&i.Name,
		&i.CreateAt,
		&i.StartAt,
		&i.StopAt,
		&i.Info,
	)
	return &i, err
}

const getAdjFactors = `-- name: GetAdjFactors :many
select id, sid, sub_id, start_ms, factor from adj_factors
where sid=$1
order by start_ms
`

func (q *Queries) GetAdjFactors(ctx context.Context, sid int32) ([]*AdjFactor, error) {
	rows, err := q.db.Query(ctx, getAdjFactors, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*AdjFactor{}
	for rows.Next() {
		var i AdjFactor
		if err := rows.Scan(
			&i.ID,
			&i.Sid,
			&i.SubID,
			&i.StartMs,
			&i.Factor,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getAllInsKlines = `-- name: GetAllInsKlines :many
select id, sid, timeframe, start_ms, stop_ms from ins_kline
`

func (q *Queries) GetAllInsKlines(ctx context.Context) ([]*InsKline, error) {
	rows, err := q.db.Query(ctx, getAllInsKlines)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*InsKline{}
	for rows.Next() {
		var i InsKline
		if err := rows.Scan(
			&i.ID,
			&i.Sid,
			&i.Timeframe,
			&i.StartMs,
			&i.StopMs,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getExOrders = `-- name: GetExOrders :many
select id, task_id, inout_id, symbol, enter, order_type, order_id, side, create_at, price, average, amount, filled, status, fee, fee_type, update_at from exorder
where inout_id=$1
`

func (q *Queries) GetExOrders(ctx context.Context, inoutID int32) ([]*ExOrder, error) {
	rows, err := q.db.Query(ctx, getExOrders, inoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*ExOrder{}
	for rows.Next() {
		var i ExOrder
		if err := rows.Scan(
			&i.ID,
			&i.TaskID,
			&i.InoutID,
			&i.Symbol,
			&i.Enter,
			&i.OrderType,
			&i.OrderID,
			&i.Side,
			&i.CreateAt,
			&i.Price,
			&i.Average,
			&i.Amount,
			&i.Filled,
			&i.Status,
			&i.Fee,
			&i.FeeType,
			&i.UpdateAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getIOrder = `-- name: GetIOrder :one
select id, task_id, symbol, sid, timeframe, short, status, enter_tag, init_price, quote_cost, exit_tag, leverage, enter_at, exit_at, strategy, stg_ver, max_pft_rate, max_draw_down, profit_rate, profit, info from iorder
where id = $1
`

func (q *Queries) GetIOrder(ctx context.Context, id int64) (*IOrder, error) {
	row := q.db.QueryRow(ctx, getIOrder, id)
	var i IOrder
	err := row.Scan(
		&i.ID,
		&i.TaskID,
		&i.Symbol,
		&i.Sid,
		&i.Timeframe,
		&i.Short,
		&i.Status,
		&i.EnterTag,
		&i.InitPrice,
		&i.QuoteCost,
		&i.ExitTag,
		&i.Leverage,
		&i.EnterAt,
		&i.ExitAt,
		&i.Strategy,
		&i.StgVer,
		&i.MaxPftRate,
		&i.MaxDrawDown,
		&i.ProfitRate,
		&i.Profit,
		&i.Info,
	)
	return &i, err
}

const getInsKline = `-- name: GetInsKline :one
select id, sid, timeframe, start_ms, stop_ms from ins_kline
where sid=$1
`

func (q *Queries) GetInsKline(ctx context.Context, sid int32) (*InsKline, error) {
	row := q.db.QueryRow(ctx, getInsKline, sid)
	var i InsKline
	err := row.Scan(
		&i.ID,
		&i.Sid,
		&i.Timeframe,
		&i.StartMs,
		&i.StopMs,
	)
	return &i, err
}

const getKHoles = `-- name: GetKHoles :many
select id, sid, timeframe, start, stop from khole
where sid = $1 and timeframe = $2
`

type GetKHolesParams struct {
	Sid       int32  `json:"sid"`
	Timeframe string `json:"timeframe"`
}

func (q *Queries) GetKHoles(ctx context.Context, arg GetKHolesParams) ([]*KHole, error) {
	rows, err := q.db.Query(ctx, getKHoles, arg.Sid, arg.Timeframe)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*KHole{}
	for rows.Next() {
		var i KHole
		if err := rows.Scan(
			&i.ID,
			&i.Sid,
			&i.Timeframe,
			&i.Start,
			&i.Stop,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTask = `-- name: GetTask :one
select id, mode, name, create_at, start_at, stop_at, info from bottask
where id = $1
`

func (q *Queries) GetTask(ctx context.Context, id int64) (*BotTask, error) {
	row := q.db.QueryRow(ctx, getTask, id)
	var i BotTask
	err := row.Scan(
		&i.ID,
		&i.Mode,
		&i.Name,
		&i.CreateAt,
		&i.StartAt,
		&i.StopAt,
		&i.Info,
	)
	return &i, err
}

const getTaskPairs = `-- name: GetTaskPairs :many
select distinct symbol from iorder
where task_id=$1 and enter_at>=$2 and enter_at<=$3
`

type GetTaskPairsParams struct {
	TaskID    int32 `json:"task_id"`
	EnterAt   int64 `json:"enter_at"`
	EnterAt_2 int64 `json:"enter_at_2"`
}

func (q *Queries) GetTaskPairs(ctx context.Context, arg GetTaskPairsParams) ([]string, error) {
	rows, err := q.db.Query(ctx, getTaskPairs, arg.TaskID, arg.EnterAt, arg.EnterAt_2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		items = append(items, symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listExchanges = `-- name: ListExchanges :many
select distinct exchange from exsymbol
`

func (q *Queries) ListExchanges(ctx context.Context) ([]string, error) {
	rows, err := q.db.Query(ctx, listExchanges)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var exchange string
		if err := rows.Scan(&exchange); err != nil {
			return nil, err
		}
		items = append(items, exchange)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listKHoles = `-- name: ListKHoles :many
select id, sid, timeframe, start, stop from khole
WHERE sid = ANY($1::int[])
`

func (q *Queries) ListKHoles(ctx context.Context, dollar_1 []int32) ([]*KHole, error) {
	rows, err := q.db.Query(ctx, listKHoles, dollar_1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*KHole{}
	for rows.Next() {
		var i KHole
		if err := rows.Scan(
			&i.ID,
			&i.Sid,
			&i.Timeframe,
			&i.Start,
			&i.Stop,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listKInfos = `-- name: ListKInfos :many
select sid, timeframe, start, stop from kinfo
`

func (q *Queries) ListKInfos(ctx context.Context) ([]*KInfo, error) {
	rows, err := q.db.Query(ctx, listKInfos)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*KInfo{}
	for rows.Next() {
		var i KInfo
		if err := rows.Scan(
			&i.Sid,
			&i.Timeframe,
			&i.Start,
			&i.Stop,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listSymbols = `-- name: ListSymbols :many
select id, exchange, exg_real, market, symbol, combined, list_ms, delist_ms from exsymbol
where exchange = $1
order by id
`

func (q *Queries) ListSymbols(ctx context.Context, exchange string) ([]*ExSymbol, error) {
	rows, err := q.db.Query(ctx, listSymbols, exchange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*ExSymbol{}
	for rows.Next() {
		var i ExSymbol
		if err := rows.Scan(
			&i.ID,
			&i.Exchange,
			&i.ExgReal,
			&i.Market,
			&i.Symbol,
			&i.Combined,
			&i.ListMs,
			&i.DelistMs,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTaskPairs = `-- name: ListTaskPairs :many
select symbol from iorder
where task_id = $1
and enter_at >= $2
and enter_at <= $3
`

type ListTaskPairsParams struct {
	TaskID    int32 `json:"task_id"`
	EnterAt   int64 `json:"enter_at"`
	EnterAt_2 int64 `json:"enter_at_2"`
}

func (q *Queries) ListTaskPairs(ctx context.Context, arg ListTaskPairsParams) ([]string, error) {
	rows, err := q.db.Query(ctx, listTaskPairs, arg.TaskID, arg.EnterAt, arg.EnterAt_2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		items = append(items, symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTasks = `-- name: ListTasks :many
select id, mode, name, create_at, start_at, stop_at, info from bottask
order by id
`

func (q *Queries) ListTasks(ctx context.Context) ([]*BotTask, error) {
	rows, err := q.db.Query(ctx, listTasks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*BotTask{}
	for rows.Next() {
		var i BotTask
		if err := rows.Scan(
			&i.ID,
			&i.Mode,
			&i.Name,
			&i.CreateAt,
			&i.StartAt,
			&i.StopAt,
			&i.Info,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const setExOrder = `-- name: SetExOrder :exec
update exorder set
       "task_id" = $1,
       "inout_id" = $2,
       "symbol" = $3,
       "enter" = $4,
       "order_type" = $5,
       "order_id" = $6,
       "side" = $7,
       "create_at" = $8,
       "price" = $9,
       "average" = $10,
       "amount" = $11,
       "filled" = $12,
       "status" = $13,
       "fee" = $14,
       "fee_type" = $15,
       "update_at" = $16
    where id = $17
`

type SetExOrderParams struct {
	TaskID    int32   `json:"task_id"`
	InoutID   int32   `json:"inout_id"`
	Symbol    string  `json:"symbol"`
	Enter     bool    `json:"enter"`
	OrderType string  `json:"order_type"`
	OrderID   string  `json:"order_id"`
	Side      string  `json:"side"`
	CreateAt  int64   `json:"create_at"`
	Price     float64 `json:"price"`
	Average   float64 `json:"average"`
	Amount    float64 `json:"amount"`
	Filled    float64 `json:"filled"`
	Status    int16   `json:"status"`
	Fee       float64 `json:"fee"`
	FeeType   string  `json:"fee_type"`
	UpdateAt  int64   `json:"update_at"`
	ID        int64   `json:"id"`
}

func (q *Queries) SetExOrder(ctx context.Context, arg SetExOrderParams) error {
	_, err := q.db.Exec(ctx, setExOrder,
		arg.TaskID,
		arg.InoutID,
		arg.Symbol,
		arg.Enter,
		arg.OrderType,
		arg.OrderID,
		arg.Side,
		arg.CreateAt,
		arg.Price,
		arg.Average,
		arg.Amount,
		arg.Filled,
		arg.Status,
		arg.Fee,
		arg.FeeType,
		arg.UpdateAt,
		arg.ID,
	)
	return err
}

const setIOrder = `-- name: SetIOrder :exec
update iorder set
      "task_id" = $1,
      "symbol" = $2,
      "sid" = $3,
      "timeframe" = $4,
      "short" = $5,
      "status" = $6,
      "enter_tag" = $7,
      "init_price" = $8,
      "quote_cost" = $9,
      "exit_tag" = $10,
      "leverage" = $11,
      "enter_at" = $12,
      "exit_at" = $13,
      "strategy" = $14,
      "stg_ver" = $15,
      "max_pft_rate" = $16,
      "max_draw_down" = $17,
      "profit_rate" = $18,
      "profit" = $19,
      "info" = $20
    WHERE id = $21
`

type SetIOrderParams struct {
	TaskID      int32   `json:"task_id"`
	Symbol      string  `json:"symbol"`
	Sid         int32   `json:"sid"`
	Timeframe   string  `json:"timeframe"`
	Short       bool    `json:"short"`
	Status      int16   `json:"status"`
	EnterTag    string  `json:"enter_tag"`
	InitPrice   float64 `json:"init_price"`
	QuoteCost   float64 `json:"quote_cost"`
	ExitTag     string  `json:"exit_tag"`
	Leverage    float64 `json:"leverage"`
	EnterAt     int64   `json:"enter_at"`
	ExitAt      int64   `json:"exit_at"`
	Strategy    string  `json:"strategy"`
	StgVer      int32   `json:"stg_ver"`
	MaxPftRate  float64 `json:"max_pft_rate"`
	MaxDrawDown float64 `json:"max_draw_down"`
	ProfitRate  float64 `json:"profit_rate"`
	Profit      float64 `json:"profit"`
	Info        string  `json:"info"`
	ID          int64   `json:"id"`
}

func (q *Queries) SetIOrder(ctx context.Context, arg SetIOrderParams) error {
	_, err := q.db.Exec(ctx, setIOrder,
		arg.TaskID,
		arg.Symbol,
		arg.Sid,
		arg.Timeframe,
		arg.Short,
		arg.Status,
		arg.EnterTag,
		arg.InitPrice,
		arg.QuoteCost,
		arg.ExitTag,
		arg.Leverage,
		arg.EnterAt,
		arg.ExitAt,
		arg.Strategy,
		arg.StgVer,
		arg.MaxPftRate,
		arg.MaxDrawDown,
		arg.ProfitRate,
		arg.Profit,
		arg.Info,
		arg.ID,
	)
	return err
}

const setKHole = `-- name: SetKHole :exec
update khole set start = $2, stop = $3
where id = $1
`

type SetKHoleParams struct {
	ID    int64 `json:"id"`
	Start int64 `json:"start"`
	Stop  int64 `json:"stop"`
}

func (q *Queries) SetKHole(ctx context.Context, arg SetKHoleParams) error {
	_, err := q.db.Exec(ctx, setKHole, arg.ID, arg.Start, arg.Stop)
	return err
}

const setKInfo = `-- name: SetKInfo :exec
update kinfo set start = $3, stop = $4
where sid = $1 and timeframe = $2
`

type SetKInfoParams struct {
	Sid       int32  `json:"sid"`
	Timeframe string `json:"timeframe"`
	Start     int64  `json:"start"`
	Stop      int64  `json:"stop"`
}

func (q *Queries) SetKInfo(ctx context.Context, arg SetKInfoParams) error {
	_, err := q.db.Exec(ctx, setKInfo,
		arg.Sid,
		arg.Timeframe,
		arg.Start,
		arg.Stop,
	)
	return err
}

const setListMS = `-- name: SetListMS :exec
update exsymbol set list_ms = $2, delist_ms = $3
where id = $1
`

type SetListMSParams struct {
	ID       int32 `json:"id"`
	ListMs   int64 `json:"list_ms"`
	DelistMs int64 `json:"delist_ms"`
}

func (q *Queries) SetListMS(ctx context.Context, arg SetListMSParams) error {
	_, err := q.db.Exec(ctx, setListMS, arg.ID, arg.ListMs, arg.DelistMs)
	return err
}
