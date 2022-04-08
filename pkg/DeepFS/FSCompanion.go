package DeepFS

import "io/fs"

type FSCloseReadDir interface {
	fs.FS
	Close() error
	ReadDir(name string) ([]fs.DirEntry, error)
}
