package tui

import (
	"fmt"
	"io"
)

// ChannelWriter implements io.Writer and sends the written data to a channel.
type ChannelWriter struct {
	Ch chan string
}

var _ io.WriteCloser = (*ChannelWriter)(nil)

func (w *ChannelWriter) GetReaderChan() <-chan string {
	return w.Ch
}

func (w *ChannelWriter) Close() error {
	// close logChan
	close(w.Ch)
	for value := range w.GetReaderChan() {
		fmt.Println(value)
	}
	return nil
}

// Write sends the byte slice as a string to the channel.
func (w *ChannelWriter) Write(p []byte) (n int, err error) {
	w.Ch <- string(p)
	return len(p), nil
}
