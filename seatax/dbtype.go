package seatax

// DBType 数据库类型。
type DBType string

const (
	// DBTypeMySQL MySQL。
	DBTypeMySQL DBType = "mysql"
	// DBTypePostgres PostgreSQL。
	DBTypePostgres DBType = "postgres"
)

// String 返回数据库类型的字符串表示。
func (t DBType) String() string { return string(t) }

// Valid 报告该数据库类型是否受支持。
func (t DBType) Valid() bool {
	switch t {
	case DBTypeMySQL, DBTypePostgres:
		return true
	default:
		return false
	}
}

// Mode 区分 RM 事务模式。
type Mode int

const (
	// ModeAT AT 模式,依赖 undo_log 表。
	ModeAT Mode = iota
	// ModeXA XA 模式,要求数据库支持 XA 协议。
	ModeXA
)

// String 返回事务模式的字符串表示。
func (m Mode) String() string {
	switch m {
	case ModeAT:
		return "AT"
	case ModeXA:
		return "XA"
	default:
		return "Unknown"
	}
}
