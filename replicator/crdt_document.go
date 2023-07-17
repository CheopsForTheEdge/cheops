package replicator

import "sort"

type crdtDocument struct {
	Locations  []string
	Generation uint64
	Payload    string
}

// sort sorts a slice of Document with a stable order: if two nodes have
// the same slice of docs, the ordering will always be the same
func sortDocuments(docs []crdtDocument) {
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Generation < docs[j].Generation {
			return true
		} else if docs[i].Generation > docs[j].Generation {
			return false
		} else {
			return sort.StringsAreSorted([]string{docs[i].Payload, docs[j].Payload})
		}
	})
}
