package ds

func SliceChunk[T any](s []T, size int) [][]T {
	chunks := [][]T{}
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}
