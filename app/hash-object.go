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
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening file %s, err: %s", path, err.Error())
		os.Exit(1)
	}
	size := strconv.Itoa(len(data))
	if err != nil {

	}
	hashedBytes := sha1.Sum(data)
	hash := hex.EncodeToString(hashedBytes[:]) // converting the fixed size [20] array to a slice
	fmt.Println(*write)
	// write flag is optional so only write the content to the object directory if enabled
	if *write {
		directory := hash[:2]
		filename := hash[2:]
		directoryPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory)
		// make directory if it doesn't already exist
		if err := os.MkdirAll(directoryPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "error creating object directory: %s, err: %s\n", directoryPath, err.Error())
			os.Exit(1)
		}
		targetPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory, filename)

		writeData := []byte{}
		writeData = append(writeData, []byte(*objectType)...)
		writeData = append(writeData, []byte(size)...)
		writeData = append(writeData, 0)
		writeData = append(writeData, data...)
		//  blob <size>\0
		if err := os.WriteFile(targetPath, writeData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing object to file, err: %s\n", err.Error())
			os.Exit(1)
		}
	}
	fmt.Fprintf(os.Stdout, "%s\n", hash)
}
