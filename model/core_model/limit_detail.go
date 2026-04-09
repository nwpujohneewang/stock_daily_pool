package core_model

type LimitDetail struct {
	FirstTime  string // 首次封板时间 "09:31:05"
	LastTime   string // 最后封板时间 "14:55:00"
	LimitTimes int    // 连板数
}
