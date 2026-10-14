package kv

const (
	OpGet    = "Get"
	OpPut    = "Put"
	OpAppend = "Append"
	OpDelete = "Delete"
)

const (
	ErrWrongLeader = "wrong leader"
	ErrTimeout     = "timeout"
	ErrLostLeader  = "leadership lost"
	ErrUnknownOp   = "unknown op"
)

type Op struct {
	Type     string
	Key      string
	Value    string
	ClientID int64
	Seq      int64
}

type Result struct {
	Value string
	Ok    bool
	Err   string
}