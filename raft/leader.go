package raft

import "time"

func (rf *Raft) becomeLeader() {
	rf.state = Leader
	for _, p := range rf.peers {
		rf.nextIndex[p] = len(rf.log)
		rf.matchIndex[p] = 0
	}
	go rf.heartbeatLoop(rf.currentTerm)
}

func (rf *Raft) heartbeatLoop(term int) {
	for {
		rf.mu.Lock()
		if rf.dead || rf.state != Leader || rf.currentTerm != term {
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
		for _, p := range rf.peers {
			if p != rf.id {
				go rf.sendAppend(p, term)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (rf *Raft) sendAppend(p, term int) {
	rf.mu.Lock()
	if rf.dead || rf.state != Leader || rf.currentTerm != term {
		rf.mu.Unlock()
		return
	}
	next := rf.nextIndex[p]
	prevIndex := next - 1
	entries := make([]LogEntry, len(rf.log)-next)
	copy(entries, rf.log[next:])
	args := &AppendEntriesArgs{
		Term:         term,
		LeaderID:     rf.id,
		PrevLogIndex: prevIndex,
		PrevLogTerm:  rf.log[prevIndex].Term,
		Entries:      entries,
		LeaderCommit: rf.commitIndex,
	}
	rf.mu.Unlock()

	var reply AppendEntriesReply
	if !rf.transport.AppendEntries(p, args, &reply) {
		return
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if reply.Term > rf.currentTerm {
		rf.becomeFollower(reply.Term)
		return
	}
	if rf.state != Leader || rf.currentTerm != term {
		return
	}
	if reply.Success {
		match := prevIndex + len(entries)
		if match > rf.matchIndex[p] {
			rf.matchIndex[p] = match
		}
		rf.nextIndex[p] = rf.matchIndex[p] + 1
		rf.advanceCommit()
	} else if rf.nextIndex[p] == next && next > 1 {
		rf.nextIndex[p] = next - 1
	}
}