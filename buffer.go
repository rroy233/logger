package logger

const MaxBufferSize = 20

type buffer struct {
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
	data := <-b.data
	return data
}
