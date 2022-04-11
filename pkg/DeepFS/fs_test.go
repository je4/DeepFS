package DeepFS

import (
	"os"
	"testing"
	"time"

	"github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
)

func TestNewDeepFS(t *testing.T) {
	log := CreateLogger("", logging.DEBUG.String(), "ZipWeb")
	zfsf := &ZipFSFactory{}
	bfs := os.DirFS("../../testdata")
	_, err := NewDeepFS(bfs, log, 500, time.Minute*3, zfsf)
	if err != nil {
		t.Fatalf("cannot create new DeepFS: %v", err)
	}
}

func TestPathFinder(t *testing.T) {
	log := CreateLogger("", logging.DEBUG.String(), "ZipWeb")
	zfsf := &ZipFSFactory{}
	bfs := os.DirFS("../../testdata")
	nfs, err := NewDeepFS(bfs, log, 500, time.Minute*3, zfsf)
	if err != nil {
		t.Fatalf("cannot create new DeepFS: %v", err)
	}
	fswc, path1, path2, err := nfs.pathFinder("test1.zip/test/sub1")
	if err != nil {
		t.Fatalf("cannot find path: %v", err)
	}
	log.Infof("fsWithCounter: %v, path1: %s, path2: %s", fswc, path1, path2)
	assert.Equal(t, path1, "test1.zip")
	assert.Equal(t, path2, "test/sub1")
}

func TestReadDir(t *testing.T) {
	log := CreateLogger("", logging.DEBUG.String(), "ZipWeb")
	zfsf := &ZipFSFactory{}
	bfs := os.DirFS("../../testdata")
	nfs, err := NewDeepFS(bfs, log, 500, time.Minute*3, zfsf)
	if err != nil {
		t.Fatalf("cannot create new DeepFS: %v", err)
	}
	dirEntries, err := nfs.ReadDir("test")
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}
	assert.Equal(t, len(dirEntries), 2)
	assert.Equal(t, dirEntries[0].Name(), "sub1")
	assert.Equal(t, dirEntries[1].Name(), "sub2")
}
