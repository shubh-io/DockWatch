package tui

import (
	"io"
	"log"
	"os"
)

var (
	debugLogger *log.Logger
	debugFile   *os.File
)

func init() {
	// default debug file in working directory
	_ = SetDebugFile("dockmate-debug.log")
}

func SetDebugFile(path string) error {
	if debugFile != nil {
		_ = debugFile.Close()
		debugFile = nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// fallback to discard
		debugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
		return err
	}
	debugFile = f
	debugLogger = log.New(debugFile, "DEBUG: ", log.LstdFlags)
	return nil
}

func CloseDebug() error {
	if debugFile == nil {
		return nil
	}
	err := debugFile.Close()
	debugFile = nil
	debugLogger = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
	return err
}
