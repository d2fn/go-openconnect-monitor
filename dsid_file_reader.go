package main

import (
	"os"
)

type DSIDFileReader struct {
	file string
}

func NewDSIDFileReader(tmpFile string) *DSIDFileReader {
	return &DSIDFileReader { file: tmpFile }
}

func (fp *DSIDFileReader) ReadDSID() (string, error) {
	bytes, err := os.ReadFile(fp.file)
	return string(bytes), err
}

