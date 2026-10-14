package raft

import (
	"bytes"
	"encoding/gob"
)

type persistedState struct {
	CurrentTerm int
	VotedFor    int
	Log         []LogEntry
}

func (rf *Raft) persist() {
	if rf.storage == nil || rf.dead {
		return
	}
	var buf bytes.Buffer
	st := persistedState{CurrentTerm: rf.currentTerm, VotedFor: rf.votedFor, Log: rf.log}
	if err := gob.NewEncoder(&buf).Encode(st); err != nil {
		panic(err)
	}
	if err := rf.storage.Save(buf.Bytes()); err != nil {
		panic(err)
	}
}

func (rf *Raft) restore() {
	if rf.storage == nil {
		return
	}
	data, err := rf.storage.Load()
	if err != nil {
		panic(err)
	}
	if len(data) == 0 {
		return
	}
	var st persistedState
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&st); err != nil {
		panic(err)
	}
	rf.currentTerm = st.CurrentTerm
	rf.votedFor = st.VotedFor
	rf.log = st.Log
}

func (rf *Raft) Kill() {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.dead = true
}