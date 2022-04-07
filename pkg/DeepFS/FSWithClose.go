package DeepFS

import "io/fs"

type FSWithClose interface {
	fs.FS
	Close() error
}
