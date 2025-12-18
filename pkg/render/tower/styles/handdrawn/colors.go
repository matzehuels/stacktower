package handdrawn

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
)

const (
	greyMin = 160
	greyMax = 250
)

func greyForID(id string) string {
	h := hash(id, 0)
	v := greyMin + int(h%uint64(greyMax-greyMin))
	return fmt.Sprintf("#%02x%02x%02x", v, v, v)
}

func hash(s string, seed uint64) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	if seed != 0 {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], seed)
		h.Write(buf[:])
	}
	return h.Sum64()
}
