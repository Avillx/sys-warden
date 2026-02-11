package main

import "errors"

var (
	FileReadError          = errors.New("Errors with read file")
	InotifyInitError       = errors.New("Can't init intify")
	InotifyAddWatcherError = errors.New("Can't add inotify watcher")
)
