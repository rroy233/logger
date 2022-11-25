package logger

import (
	"sync"
)

const MaxBufferSize = 10

type buffer struct {
	lock sync.Mutex
	data chan string
}

func NewBuffer() *buffer {
	b := new(buffer)
	b.data = make(chan string, MaxBufferSize)
	return b
}

func (b *buffer) Write(p []byte) (int, error) {
	b.data <- string(p)[:len(p)-1]
	return len(p), nil
}

func (b *buffer) GetOne() string {
	b.lock.Lock()
	data := <-b.data
	b.lock.Unlock()
	return data
}
