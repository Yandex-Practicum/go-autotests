package random

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	mathrand "math/rand"
)

// rnd generates new random generator with new source for each binary call
var rnd = func() *mathrand.Rand {
	buf := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(err)
	}
	src := mathrand.NewSource(int64(binary.LittleEndian.Uint64(buf)))
	return mathrand.New(src)
}()
