package fork

import (
	"bytes"
	"sync"
)

type buffer struct {
	m   sync.RWMutex
	buf bytes.Buffer
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf.Write(p)
}

func (b *buffer) Read(p []byte) (n int, err error) {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.buf.Read(p)
}

func (b *buffer) Bytes() []byte {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.buf.Bytes()
}
