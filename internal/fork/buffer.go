package fork

import (
	"bytes"
	"sync"
)

// buffer вяляется синхронной оберткой над bytes.Buffer
type buffer struct {
	m   sync.RWMutex
	buf bytes.Buffer
}

// Write реализует интерфейс io.Writer
func (b *buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buf.Write(p)
}

// Read реализует интерфейс io.Reader
func (b *buffer) Read(p []byte) (n int, err error) {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.buf.Read(p)
}

// Bytes возвращает все байты из буфера
func (b *buffer) Bytes() []byte {
	b.m.RLock()
	defer b.m.RUnlock()
	return b.buf.Bytes()
}
