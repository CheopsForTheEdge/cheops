package replicator

import (
	"bytes"
	"sort"

	"github.com/anacrolix/torrent/bencode"
)

type crdtDocument struct {
	Id         string `json:"_id,omitempty"`  // Only used by couchdb
	Rev        string `json:"_rev,omitempty"` // Only used by couchdb
	Locations  []string
	Generation uint64
	Payload    Payload
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
			iEncoded, erri := bencode.Marshal(docs[i].Payload)
			jEncoded, errj := bencode.Marshal(docs[j].Payload)
			if erri != nil || errj != nil {
				return true
			}
			return bytes.Compare(iEncoded, jEncoded) <= 0
		}
	})
}
