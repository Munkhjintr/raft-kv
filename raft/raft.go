package raft

import (
	"sync"
	"time"
)

type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	}
	return "Unknown"
}

type LogEntry struct {
	Term    int
	Command interface{}
}


type Raft struct {
	mu    sync.Mutex
	id    int
	peers []int


	currentTerm int
	votedFor    int
	log         []LogEntry


	state       State
	commitIndex int
	lastApplied int


	nextIndex  map[int]int
	matchIndex map[int]int

	lastHeartbeat time.Time
}

func New(id int, peers []int) *Raft {
	return &Raft{
		id:            id,
		peers:         peers,
		votedFor:      -1,
		log:           []LogEntry{{Term: 0}},
		state:         Follower,
		nextIndex:     make(map[int]int),
		matchIndex:    make(map[int]int),
		lastHeartbeat: time.Now(),
	}
}

func (rf *Raft) GetState() (term int, isLeader bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.state == Leader
}
