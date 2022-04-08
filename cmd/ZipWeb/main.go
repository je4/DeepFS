package main

import (
	"context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/je4/DeepFS/v2/pkg/DeepFS"
	"github.com/je4/ZipFS/v2/pkg/ZipFS"
	"github.com/op/go-logging"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ZipFSFactory struct{}

func (zfsf *ZipFSFactory) GetExtension() string { return ".zip" }

func (zfsf *ZipFSFactory) CreateFS(parent fs.FS, path string) (DeepFS.FSCloseReadDir, error) {
	return ZipFS.NewZipFS(parent, path)
}

func main() {
	var _logformat = logging.MustStringFormatter(
		`%{time:2006-01-02T15:04:05.000} %{module}::%{shortfunc} [%{shortfile}] > %{level:.5s} - %{message}`,
	)
	var log *logging.Logger
	var lf *os.File
	log = logging.MustGetLogger("ZipWeb")
	var err error
	var logfile = ""
	var loglevel = "DEBUG"
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

	zfsf := &ZipFSFactory{}
	bfs := os.DirFS("C:/temp/")
	nfs, err := DeepFS.NewDeepFS(bfs, log, time.Minute*3, zfsf)
	if err != nil {
		log.Panic(err)
	}
	fileServer := http.FileServer(http.FS(nfs))

	log.Infof("http://localhost:8080/")

	router := mux.NewRouter()
	router.PathPrefix("/").Handler(fileServer)
	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, handlers.ProxyHeaders(router))

	srv := &http.Server{
		Handler: loggedRouter,
		Addr:    ":8080",
	}
	go func() {
		if err = srv.ListenAndServe(); err != nil {
			log.Fatalf("server died: %v", err)
		}
	}()

	end := make(chan bool, 1)

	// process waiting for interrupt signal (TERM or KILL)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)

		signal.Notify(sigint, syscall.SIGTERM)
		signal.Notify(sigint, syscall.SIGKILL)

		<-sigint

		// We received an interrupt signal, shut down.
		log.Infof("shutdown requested")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		srv.Shutdown(ctx)

		end <- true
	}()

	<-end
	log.Info("server stopped")
}
