package replicator

import (
	"bytes"
	"log"
	"sort"

	"github.com/anacrolix/torrent/bencode"
	jp "github.com/evanphx/json-patch"
)

type crdtDocument struct {
	Id  string `json:"_id,omitempty"`
	Rev string `json:"_rev,omitempty"`

	Locations  []string
	Generation uint64
	Payload    Payload

	// When true for a request, it means the intent is for this document to not exist anymore
	// When true for a reply, it means the deletion has been processed
	Deleted bool
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

// mergePatches takes all requests in a given order and
// produces a document that represents a unification of all
// requests. The resulting document can be applied as-is
func mergePatches(requests []crdtDocument) []byte {
	b := []byte("{}")
	var err error
	for _, r := range requests {
		b, err = jp.MergeMergePatches(b, []byte(r.Payload.Body))
		if err != nil {
			log.Println("Couldn't merge patches")
			// Not actually problematic, continue
		}
	}

	return b
}

// MetaDocument are documents stored in the cheops-all database
type MetaDocument struct {

	// can be SITE or RESOURCE
	Type string

	// if type == SITE or type == RESOURCE
	Site string

	// if type == RESOURCE
	ResourceId string `json:",omitempty"`
}
