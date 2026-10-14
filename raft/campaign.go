package raft

import "time"

func (rf *Raft) ticker() {
	for {
		time.Sleep(10 * time.Millisecond)
		rf.mu.Lock()
		if rf.dead {
			rf.mu.Unlock()
			return
		}
		if rf.state != Leader && time.Since(rf.lastHeartbeat) >= rf.electionTimeout {
			rf.startElection()
		}
		rf.mu.Unlock()
	}
}

func (rf *Raft) startElection() {
	rf.state = Candidate
	rf.currentTerm++
	rf.votedFor = rf.id
	rf.persist()
	rf.lastHeartbeat = time.Now()
	rf.resetElectionTimeout()

	term := rf.currentTerm
	lastIndex, lastTerm := rf.lastLogInfo()
	votes := 1

	for _, p := range rf.peers {
		if p == rf.id {
			continue
		}
				go rf.requestVoteFrom(p, term, lastIndex, lastTerm, &votes)
	}
	if votes > len(rf.peers)/2 {
		rf.becomeLeader()
	}
}

func (rf *Raft) requestVoteFrom(p, term, lastIndex, lastTerm int, votes *int) {
	args := &RequestVoteArgs{
		Term:         term,
		CandidateID:  rf.id,
		LastLogIndex: lastIndex,
		LastLogTerm:  lastTerm,
	}
	var reply RequestVoteReply
	if !rf.transport.RequestVote(p, args, &reply) {
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if reply.Term > rf.currentTerm {
		rf.becomeFollower(reply.Term)
		return
	}
	if rf.state != Candidate || rf.currentTerm != term || !reply.VoteGranted {
		return
	}
	*votes++
	if *votes > len(rf.peers)/2 {
		rf.becomeLeader()
	}
}