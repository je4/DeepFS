package DeepFS

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bluele/gcache"
	"github.com/je4/ZipFS/v2/pkg/ZipFS"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

// /data/abc/test.zip/content/image.png
type DeepFS struct {
	subFS   map[string]FSFactory
	fsCache gcache.Cache
	baseFS  fs.FS
	log     *logging.Logger
}

func CreateLogger(logFileName string, logLevel string, loggerName string) *logging.Logger {
	var _logformat = logging.MustStringFormatter(
		`%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} [%{shortfile}] > %{level:.5s} - %{message}`,
	)
	var log *logging.Logger
	var lf *os.File
	log = logging.MustGetLogger(loggerName)
	var err error
	var logfile = logFileName
	var loglevel = logLevel
	if logfile != "" {
		lf, err = os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Errorf("Cannot open logfile %v: %v", logfile, err)
		}
		//defer lf.CloseInternal()
	} else {
		lf = os.Stderr
	}
	backend := logging.NewLogBackend(lf, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.GetLevel(loglevel), "")

	logging.SetFormatter(_logformat)
	logging.SetBackend(backendLeveled)

	return log
}

func NewDeepFS(baseFS fs.FS, log *logging.Logger, cacheSize int, cacheTimeout time.Duration, fss ...FSFactory) (*DeepFS, error) {
	dfs := &DeepFS{
		baseFS: baseFS,
		subFS:  map[string]FSFactory{},
		log:    log,
	}
	dfs.fsCache = gcache.New(cacheSize).
		LRU().
		Expiration(cacheTimeout).
		EvictedFunc(func(key, value interface{}) {
			ext, ok := key.(string)
			if !ok {
				log.Errorf("invalid key type: %v", key)
				return
			}
			log.Infof("removing %s from cache", key)
			fsc, ok := value.(fsWithCounter)
			if !ok {
				log.Errorf("invalid type of cached filesystem for %s", ext)
			}
			if fsc.hasOpenFiles() {
				log.Infof("cached filesystem for %s has open files", ext)
				if err := dfs.fsCache.Set(ext, value); err != nil {
					log.Errorf("cannot reset cache entry for %s: %v", ext, err)
				}

				return
			}
			if err := fsc.Close(); err != nil {
				log.Errorf("cannot close cached filesystem for %s: %v", ext, err)
				if err := dfs.fsCache.Set(ext, value); err != nil {
					log.Errorf("cannot reset cache entry for %s: %v", ext, err)
				}
			}
		}).Build()

	for _, sfs := range fss {
		ext := strings.ToLower(sfs.GetExtension())
		dfs.subFS[ext] = sfs
	}
	return dfs, nil
}

type ZipFSFactory struct{}

func (zfsf *ZipFSFactory) GetExtension() string { return ".zip" }

func (zfsf *ZipFSFactory) CreateFS(parent fs.FS, path string) (FSCloseReadDir, error) {
	return ZipFS.NewZipFS(parent, path)
}

type fsWithCounter struct {
	FSCloseReadDir
	counter int64
	log     *logging.Logger
}

func (fswc *fsWithCounter) inc() {
	fswc.log.Infof("increment counter: %d", atomic.AddInt64(&fswc.counter, 1))
}

func (fswc *fsWithCounter) dec() {
	fswc.log.Infof("decrement counter: %d", atomic.AddInt64(&fswc.counter, -1))
}

func (fswc *fsWithCounter) hasOpenFiles() bool {
	return atomic.LoadInt64(&fswc.counter) > 0
}

func (dfs *DeepFS) getFS(path, ext string) (*fsWithCounter, error) {
	fsInt, err := dfs.fsCache.Get(path)
	var f *fsWithCounter
	if err == nil {
		var ok bool
		f, ok = fsInt.(*fsWithCounter)
		if !ok {
			return nil, errors.Errorf("invalid type in cache for %s: %T", path, fsInt)
		}
	} else {
		sfs, ok := dfs.subFS[ext] // must work!!!!!
		if !ok {                  // paranoia
			return nil, errors.Errorf("invalid subFS for %s", ext)
		}
		xf, err := sfs.CreateFS(dfs.baseFS, path)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create filesystem for %s", path)
		}
		if err := dfs.fsCache.Set(path, &fsWithCounter{FSCloseReadDir: xf, log: dfs.log}); err != nil {
			return nil, errors.Wrapf(err, "cannot cache new filesystem for %s", path)
		}

	}
	return f, nil
}

func (dfs *DeepFS) pathFinder(path string) (*fsWithCounter, string, string, error) {
	parts := strings.Split(path, "/")
	for idx, part := range parts {
		ext := strings.ToLower(filepath.Ext(part))
		for sext, _ := range dfs.subFS {
			if sext == ext {
				path1 := strings.Join(parts[0:idx+1], "/")
				path2 := strings.Join(parts[idx+1:], "/")
				daFS, err := dfs.getFS(path1, ext)
				if err != nil {
					return nil, "", "", errors.Wrapf(err, "cannot get filesystem for %s", path)
				}
				return daFS, path1, path2, nil
			}
		}
	}
	return nil, "", "", nil
}

func (dfs *DeepFS) Open(name string) (fs.File, error) {
	name = filepath.ToSlash(filepath.Clean(name))
	nfs, _, path2, err := dfs.pathFinder(name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot determine path %s", name)
	}
	if nfs == nil {
		return dfs.baseFS.Open(name)
	}
	f, err := nfs.Open(path2)
	if err != nil {
		return nil, err
	}
	nfs.inc()
	return &File{File: f, daFS: nfs}, nil
}

func (dfs *DeepFS) ReadDir(name string) ([]fs.DirEntry, error) {
	name = filepath.ToSlash(filepath.Clean(name))
	nfs, _, path2, err := dfs.pathFinder(name)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot determine path %s", name)
	}
	if nfs == nil {
		return fs.ReadDir(dfs.baseFS, name)
	}
	return nfs.ReadDir(path2)
}
