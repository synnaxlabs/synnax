package binary

// MakeCopy returns a copy of the given byte slice.
func MakeCopy(bytes []byte) []byte {
	copied := make([]byte, len(bytes))
	copy(copied, bytes)
	return copied
}
