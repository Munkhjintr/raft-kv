package kv

import "encoding/gob"

func init() {
	gob.Register(Op{})
}