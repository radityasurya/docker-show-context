package client

import "sort"

// SizeAscending returns directories in m sorted by Size (biggest first).
func SizeAscending(m map[string]int64) []PathSize {
	bySize := BySize{}
	for dir, size := range m {
		bySize = append(bySize, PathSize{dir, size})
	}
	sort.Sort(bySize)
	return []PathSize(bySize)
}

// PathSize represents a directory with a size.
type PathSize struct {
	Path string
	Size int64
}

// BySize sorts by size (biggest first).
type BySize []PathSize

func (a BySize) Len() int           { return len(a) }
func (a BySize) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySize) Less(i, j int) bool { return a[i].Size > a[j].Size }
