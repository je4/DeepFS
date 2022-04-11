package DeepFS

import (
	"io/fs"

	"github.com/pkg/errors"
)

type File struct {
	fs.File
	daFS *fsWithCounter
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.File.Stat()
}

func (f *File) Read(b []byte) (int, error) {
	return f.File.Read(b)
}

func (f *File) Close() error {
	f.daFS.dec()
	return f.File.Close()
}

func (f *File) ReadDir(count int) ([]fs.DirEntry, error) {
	fwrd, ok := f.File.(fs.ReadDirFile)
	if !ok {
		return nil, errors.Errorf("%T directory missing ReadDir method", f.File)
	}
	return fwrd.ReadDir(count)
}
