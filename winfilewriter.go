package main

import (
	"os"
)

type winFileWriter struct {
	file *os.File
}

func NewWinFileWriter(file *os.File) *winFileWriter {
	return &winFileWriter{
		file: file,
	}
}

func (wfw *winFileWriter) Write(p []byte) (n int, err error) {
	n, err = wfw.file.Write(p)

	wfw.file.Sync()

	return n, err
}
