package raft

import (
	"math/rand"
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
	mu        sync.Mutex
	id        int
	peers     []int
	transport Transport

	currentTerm int
	votedFor    int
	log         []LogEntry

	state       State
	commitIndex int
	lastApplied int

	nextIndex  map[int]int
	matchIndex map[int]int

	lastHeartbeat   time.Time
	electionTimeout time.Duration
}

func New(id int, peers []int, t Transport) *Raft {
	rf := &Raft{
		id:            id,
		peers:         peers,
		transport:     t,
		votedFor:      -1,
		log:           []LogEntry{{Term: 0}},
		state:         Follower,
		nextIndex:     make(map[int]int),
		matchIndex:    make(map[int]int),
		lastHeartbeat: time.Now(),
	}
	rf.resetElectionTimeout()
	return rf
}

func (rf *Raft) resetElectionTimeout() {
	rf.electionTimeout = time.Duration(150+rand.Intn(150)) * time.Millisecond
}

func (rf *Raft) Start() {
	go rf.ticker()
}

func (rf *Raft) GetState() (term int, isLeader bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.currentTerm, rf.state == Leader
}