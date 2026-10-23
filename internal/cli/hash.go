package cli

import (
	"encoding/binary"
	"hash/fnv"

	"sixletters/back-to-the-past/internal/model"
)

// hashView returns a deterministic fingerprint of a full book view.
// It is a regression/determinism check, not a security hash — FNV-1a over a
// canonical binary encoding is sufficient and cheap.
func hashView(v model.BookView) uint64 {
	h := fnv.New64a()
	var buf [8]byte

	putU64 := func(x uint64) {
		binary.BigEndian.PutUint64(buf[:], x)
		h.Write(buf[:])
	}
	putStr := func(s string) {
		putU64(uint64(len(s)))
		h.Write([]byte(s))
	}

	putStr(v.Market.Venue)
	putStr(v.Market.Instrument)
	putU64(uint64(v.Cursor.Sequence))
	for _, lvl := range v.Bids {
		putU64(uint64(lvl.Price))
		putU64(uint64(lvl.Quantity))
	}
	for _, lvl := range v.Asks {
		putU64(uint64(lvl.Price))
		putU64(uint64(lvl.Quantity))
	}
	return h.Sum64()
}
