package buffalo

import "io"

type BufferedWriter struct {
	Buffer *Buffer
	Writer io.Writer
}

// Write implements the io.Writer interface
func (b *BufferedWriter) Write(p []byte) (n int, err error) {
	if len(p) > b.Buffer.Cap() {
		return b.Writer.Write(p)
	}
	if b.Buffer.Len()+len(p) > b.Buffer.Cap() {
		if err = b.Flush(); err != nil {
			return
		}
	}
	return b.Buffer.Write(p)
}

// Flush flushes the buffer to the underlying writer.
func (b *BufferedWriter) Flush() error {
	_, err := b.Buffer.WriteTo(b.Writer)
	if err != nil {
		return err
	}
	b.Buffer.Reset()
	return nil
}
