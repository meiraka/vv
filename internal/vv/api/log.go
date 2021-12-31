package api

type Logger interface {
	Printf(string, ...interface{})
	Println(...interface{})
	Debugf(string, ...interface{})
	Debugln(...interface{})
}
