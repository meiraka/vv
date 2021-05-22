package api

import "errors"

var (
	// ErrAlreadyShutdown returns if already Shutdown is called
	ErrAlreadyShutdown = errors.New("api: already shutdown")
	// errAlreadyUpdating returns if already Update is called
	errAlreadyUpdating = errors.New("api: update already started")
)
