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

// Raft is a single node. Fields follow Figure 2 of the Raft paper.
type Raft struct {
	mu    sync.Mutex
	id    int
	peers []int // ids of all nodes, including this one

	// Persistent state (will be saved to disk later)
	currentTerm int
	votedFor    int // -1 means none
	log         []LogEntry

	// Volatile state
	state       State
	commitIndex int
	lastApplied int

	// Volatile state on leaders (reinitialized after election)
	nextIndex  map[int]int
	matchIndex map[int]int

	lastHeartbeat time.Time
}

func New(id int, peers []int) *Raft {
	return &Raft{
		id:            id,
		peers:         peers,
		votedFor:      -1,
		log:           []LogEntry{{Term: 0}}, // dummy entry at index 0
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
