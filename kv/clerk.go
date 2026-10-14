package kv

import (
	"errors"
	"math/rand"
	"time"
)

type Clerk struct {
	servers  []*Server
	clientID int64
	seq      int64
	leader   int
}

func NewClerk(servers []*Server) *Clerk {
	return &Clerk{servers: servers, clientID: rand.Int63()}
}

func (c *Clerk) do(op Op) (Result, error) {
	c.seq++
	op.ClientID = c.clientID
	op.Seq = c.seq
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		r := c.servers[c.leader].Submit(op)
		if r.Err == "" {
			return r, nil
		}
		c.leader = (c.leader + 1) % len(c.servers)
		time.Sleep(20 * time.Millisecond)
	}
	return Result{}, errors.New("request timed out")
}

func (c *Clerk) Get(key string) (string, bool, error) {
	r, err := c.do(Op{Type: OpGet, Key: key})
	return r.Value, r.Ok, err
}

func (c *Clerk) Put(key, value string) error {
	_, err := c.do(Op{Type: OpPut, Key: key, Value: value})
	return err
}

func (c *Clerk) Append(key, value string) error {
	_, err := c.do(Op{Type: OpAppend, Key: key, Value: value})
	return err
}

func (c *Clerk) Delete(key string) error {
	_, err := c.do(Op{Type: OpDelete, Key: key})
	return err
}