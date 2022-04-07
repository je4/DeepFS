package DeepFS

import "io/fs"

type FSFactory interface {
	GetExtension() string
	CreateFS(parent fs.FS, path string) (FSWithClose, error)
}
