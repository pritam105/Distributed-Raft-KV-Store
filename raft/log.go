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

func (l *logStore) lastIndex() uint64 {
	return l.entries[len(l.entries)-1].Index
}

func (l *logStore) entryAt(index uint64) (LogEntry, bool) {
	for _, e := range l.entries {
		if e.Index == index {
			return e, true
		}
	}
	return LogEntry{}, false
}

func (l *logStore) appendEntry(entry LogEntry) {
	l.entries = append(l.entries, entry)
}

// truncateAfter removes all entries with Index > index.
func (l *logStore) truncateAfter(index uint64) {
	for i, e := range l.entries {
		if e.Index > index {
			l.entries = l.entries[:i]
			return
		}
	}
}

// entriesFrom returns a copy of all entries with Index >= index.
func (l *logStore) entriesFrom(index uint64) []LogEntry {
	for i, e := range l.entries {
		if e.Index >= index {
			result := make([]LogEntry, len(l.entries)-i)
			copy(result, l.entries[i:])
			return result
		}
	}
	return nil
}
