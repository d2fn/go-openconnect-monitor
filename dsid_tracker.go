package main

type DSIDTracker struct {
	rejected map[string]int
	current  string
}

func NewDSIDTracker() *DSIDTracker {
	t := &DSIDTracker{rejected: make(map[string]int)}
	t.rejected[""]++
	return t
}

type DSIDStatus int

const (
	Rejected DSIDStatus = iota
	Accepted
	Active
)

func (t *DSIDTracker) notify(dsid string) DSIDStatus {
	if dsid == t.current {
		// this is and was the acive DSID
		return Active
	}
	if _, ok := t.rejected[dsid]; ok {
		// dsid was previously rejected
		t.rejected[dsid]++
		return Rejected
	}
	// DSID was not previously rejected and is the latest available, move the existing
	// key into the rejected pile and set this as the new active key
	t.rejected[t.current]++
	t.current = dsid
	// accept the newest dsid as the current
	return Accepted
}

func (t *DSIDTracker) reject(dsid string) {
	t.rejected[dsid]++
	if t.current == dsid {
		t.current = ""
	}
}
