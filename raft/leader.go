package raft

import "time"

func (rf *Raft) becomeLeader() {
	rf.state = Leader
	for _, p := range rf.peers {
		rf.nextIndex[p] = len(rf.log)
		rf.matchIndex[p] = 0
	}
	for _, p := range rf.peers {
		if p != rf.id {
			go rf.replicator(p, rf.currentTerm)
		}
	}
}

func (rf *Raft) notifyPeer(p int) {
	select {
	case rf.repNotify[p] <- struct{}{}:
	default:
	}
}

func (rf *Raft) kick() {
	for _, p := range rf.peers {
		if p != rf.id {
			rf.notifyPeer(p)
		}
	}
}

func (rf *Raft) replicator(p, term int) {
	for {
		rf.mu.Lock()
		if rf.dead || rf.state != Leader || rf.currentTerm != term {
			rf.mu.Unlock()
			return
		}
		ch := rf.repNotify[p]
		rf.mu.Unlock()

		rf.sendAppend(p, term)

		select {
		case <-ch:
		case <-time.After(50 * time.Millisecond):
		}
	}
}