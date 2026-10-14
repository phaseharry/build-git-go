package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func hashObject(args []string) {
	fs := flag.NewFlagSet("cat-file", flag.ExitOnError)
	objectType := fs.String("t", "blob", "type of object being hashed")
	write := fs.Bool("w", false, "write object to object directory")
	fs.Parse(args)

	// if `git hash-object` is ran with no arguments, it doesn't seem to error out.
	// there's just no output and no hash is generated
	if fs.NArg() < 1 {
		return
	}

	path := fs.Arg(0)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening file %s, err: %s", path, err.Error())
		os.Exit(1)
	}
	size := strconv.Itoa(len(contentBytes))

	writeBytes := []byte{}
	writeBytes = append(writeBytes, []byte(*objectType)...)
	writeBytes = append(writeBytes, ' ')
	writeBytes = append(writeBytes, []byte(size)...)
	writeBytes = append(writeBytes, 0)
	writeBytes = append(writeBytes, contentBytes...)

	// creates a new hasher that can take in bytes and when it has all the required bytes
	// call .Sum to get the hashed output
	h := sha1.New()
	h.Write(writeBytes)
	hashedBytes := h.Sum(nil)
	hash := hex.EncodeToString(hashedBytes)

	// write flag is optional so only write the content to the object directory if enabled
	if *write {
		directory := hash[:2]
		filename := hash[2:]

		// make directory if it doesn't already exist
		directoryPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory)
		if err := os.MkdirAll(directoryPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "error creating object directory: %s, err: %s\n", directoryPath, err.Error())
			os.Exit(1)
		}

		// compressing writeBytes before it is written to file
		var buf bytes.Buffer
		zipWriter := zlib.NewWriter(&buf)
		if _, err := zipWriter.Write(writeBytes); err != nil {
			fmt.Fprintf(os.Stderr, "error compressing content %s\n", err.Error())
			os.Exit(1)
		}
		// calling Close flushes the rest of the bytes. buf isn't complete until Close() returns
		if err := zipWriter.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error compressing content %s\n", err.Error())
			os.Exit(1)
		}

		targetPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory, filename)
		if err := os.WriteFile(targetPath, buf.Bytes(), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing object to file, err: %s\n", err.Error())
			os.Exit(1)
		}
	}
	fmt.Fprintf(os.Stdout, "%s\n", hash)
}
