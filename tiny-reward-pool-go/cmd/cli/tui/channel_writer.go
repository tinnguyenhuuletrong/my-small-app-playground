package tui

// ChannelWriter implements io.Writer and sends the written data to a channel.
type ChannelWriter struct {
	Ch chan<- string
}

// Write sends the byte slice as a string to the channel.
func (w *ChannelWriter) Write(p []byte) (n int, err error) {
	w.Ch <- string(p)
	return len(p), nil
}
