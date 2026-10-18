package main

import (
	"flag"
	"fmt"
	"os"
)

func hashObject(args []string) {
	fs := flag.NewFlagSet("hash-object", flag.ExitOnError)
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

	data, hash := encodeObject(*objectType, contentBytes)
	// creates a new hasher that can take in bytes and when it has all the required bytes
	// call .Sum to get the hashed output

	// write flag is optional so only write the content to the object directory if enabled
	if *write {
		if err := writeObject(hash, data); err != nil {
			os.Exit(1)
		}
	}
	fmt.Fprintf(os.Stdout, "%s\n", hash)
}
