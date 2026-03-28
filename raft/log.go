package raft

type LogEntry struct {
	Term    uint64
	Index   uint64
	Command []byte
}

type logStore struct {
	entries []LogEntry
}

func newLogStore() *logStore {
	return &logStore{
		entries: []LogEntry{{Term: 0, Index: 0}},
	}
}

func (l *logStore) lastIndexAndTerm() (uint64, uint64) {
	last := l.entries[len(l.entries)-1]
	return last.Index, last.Term
}
