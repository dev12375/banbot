package orm

type InOutOrder struct {
	*IOrder
	Enter      *ExOrder
	Exit       *ExOrder
	Info       map[string]interface{}
	DirtyMain  bool   // IOrder 有未保存的临时修改
	DirtyEnter bool   // Enter 有未保存的临时修改
	DirtyExit  bool   // Exit 有未保存的临时修改
	DirtyInfo  bool   // Info 有未保存的临时修改
	idKey      string // 区分订单的key
}

type InOutEdit struct {
	Order  *InOutOrder
	Action string
}

type KlineAgg struct {
	TimeFrame string
	MSecs     int64
	Table     string
	AggFrom   string
	AggStart  string
	AggEnd    string
	AggEvery  string
	CpsBefore string
	Retention string
}

type AdjFactorExt struct {
	*AdjFactor
	StopMs int64 // 13位区间结束时间戳
}
