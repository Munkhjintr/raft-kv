package raft

import "time"

func (rf *Raft) signalApply() {
	select {
	case rf.applyNotify <- struct{}{}:
	default:
	}
}

func (rf *Raft) Propose(command interface{}) (index, term int, isLeader bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.dead || rf.state != Leader {
		return -1, rf.currentTerm, false
	}
	rf.log = append(rf.log, LogEntry{Term: rf.currentTerm, Command: command})
	rf.persist()
	rf.advanceCommit()
	rf.kick()
	return len(rf.log) - 1, rf.currentTerm, true
}

func (rf *Raft) advanceCommit() {
	for n := len(rf.log) - 1; n > rf.commitIndex; n-- {
		if rf.log[n].Term != rf.currentTerm {
			break
		}
		count := 1
		for _, p := range rf.peers {
			if p != rf.id && rf.matchIndex[p] >= n {
				count++
			}
		}
		if count > len(rf.peers)/2 {
			rf.commitIndex = n
			rf.signalApply()
			rf.kick()
			break
		}
	}
}

func (rf *Raft) applier() {
	for {
		select {
		case <-rf.applyNotify:
		case <-time.After(50 * time.Millisecond):
		}
		rf.mu.Lock()
		if rf.dead {
			rf.mu.Unlock()
			return
		}
		var msgs []ApplyMsg
		for rf.lastApplied < rf.commitIndex {
			rf.lastApplied++
			msgs = append(msgs, ApplyMsg{
				Index:   rf.lastApplied,
				Command: rf.log[rf.lastApplied].Command,
			})
		}
		rf.mu.Unlock()
		for _, m := range msgs {
			rf.applyCh <- m
		}
	}
}