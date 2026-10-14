package main

import (
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

	// creates a new hasher that can take in bytes and when it has all the required bytes
	// call .Sum to get the hashed output
	h := sha1.New()
	h.Write([]byte(*objectType))
	h.Write([]byte(size))
	h.Write([]byte{0})
	h.Write(contentBytes)

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

		targetPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory, filename)
		if err := os.WriteFile(targetPath, hashedBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing object to file, err: %s\n", err.Error())
			os.Exit(1)
		}
	}
	fmt.Fprintf(os.Stdout, "%s\n", hash)
}
