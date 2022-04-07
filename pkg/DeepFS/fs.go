package DeepFS

import (
	"github.com/bluele/gcache"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// /data/abc/test.zip/content/image.png
type DeepFS struct {
	baseFS  fs.FS
	subFS   map[string]FSFactory
	fsCache gcache.Cache
	log     logging.Logger
}

/*
var _logformat = logging.MustStringFormatter(
	`%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} [%{shortfile}] > %{level:.5s} - %{message}`,
)
	var log *logging.Logger
	var lf *os.File
	log = logging.MustGetLogger(module)
	var err error
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


*/
func NewDeepFS(baseFS fs.FS, log logging.Logger, cacheTimeout time.Duration, fss ...FSFactory) (*DeepFS, error) {
	dfs := &DeepFS{
		baseFS: baseFS,
		subFS:  map[string]FSFactory{},
	}
	dfs.fsCache = gcache.New(500).
		LRU().
		Expiration(cacheTimeout).
		EvictedFunc(func(key, value interface{}) {
			ext, ok := key.(string)
			if !ok {
				log.Errorf("invalid key type: %v", key)
				return
			}
			log.Infof("removing %s from cache", key)
			fsc, ok := value.(FSWithClose)
			if !ok {
				log.Errorf("invalid type of cached filesystem for %s", ext)
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

func (dfs *DeepFS) getFS(path, ext string) (fs.FS, error) {
	fsInt, err := dfs.fsCache.Get(path)
	fs, ok := fsInt.(FSWithClose)
	if !ok {
		return nil, errors.Errorf("invalid type in cache for %s: %T", path, fsInt)
	}
	if err != nil {
		sfs, ok := dfs.subFS[ext] // must work!!!!!
		if !ok {                  // paranoia
			return nil, errors.Errorf("invalid subFS for %s", ext)
		}
		fs, err = sfs.CreateFS(dfs.baseFS, path)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot create filesystem for %s", path)
		}
		if err := dfs.fsCache.Set(path, fs); err != nil {
			return nil, errors.Wrapf(err, "cannot cache new filesystem for %s", path)
		}
	}
	return fs, nil
}

func (dfs *DeepFS) pathFinder(path string) (fs.FS, string, string, error) {
	parts := strings.Split(path, "/")
	for idx, part := range parts {
		ext := strings.ToLower(filepath.Ext(part))
		for sext, _ := range dfs.subFS {
			if sext == ext {
				path1 := strings.Join(parts[0:idx], "/")
				path2 := strings.Join(parts[idx:], "/")
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
