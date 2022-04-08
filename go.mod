module github.com/je4/DeepFS/v2

go 1.18

replace github.com/je4/DeepFS/v2 => ./

replace github.com/je4/ZipFS/v2 => ../ZipFS/

require (
	github.com/bluele/gcache v0.0.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/je4/ZipFS/v2 v2.0.0-00010101000000-000000000000
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	github.com/pkg/errors v0.9.1
)

require github.com/felixge/httpsnoop v1.0.1 // indirect
