package main

import (
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// takes a file path and opens the open to get its raw bytes and then decompress it
func getObject(hash string) ([]byte, error) {
	subDirectory, filename := hash[:2], hash[2:]
	path := filepath.Join(GIT_OBJECTS_DIRECTORY, subDirectory, filename)

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "file not found")
		return []byte{}, err
	} else if err != nil {
		fmt.Fprint(os.Stderr, "issue with opening object file\n")
		return []byte{}, err
	}
	defer file.Close()

	// using the zlib read to decompress the object's content
	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decompressing object: %s\n", err)
		return []byte{}, err
	}
	defer zlibReader.Close()

	data, err := io.ReadAll(zlibReader)
	return data, err
}
