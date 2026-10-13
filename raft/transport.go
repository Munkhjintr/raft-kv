package raft

type Transport interface {
	RequestVote(to int, args *RequestVoteArgs, reply *RequestVoteReply) bool
	AppendEntries(to int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool
}