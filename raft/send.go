package raft

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
		rf.notifyPeer(p)
	}
}