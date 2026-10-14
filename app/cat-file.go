package main

import (
	"bytes"
	"compress/zlib"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func catFile(args []string) {
	fs := flag.NewFlagSet("cat-file", flag.ExitOnError)
	pretty := fs.Bool("p", false, "pretty-print object contents")
	fs.Parse(args)

	// checks the number of arguments that are not flags after all flag values have been parsed
	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object>\n")
	}

	// flags are required for cat-file command. for now we only support -p
	if !*pretty {
		fmt.Fprint(os.Stderr, "usage: requires a flag to be set\n")
		os.Exit(1)
	}

	hash := fs.Arg(0)
	// fmt.Fprintf(os.Stdout, "pretty=%v hash=%s\n", *pretty, hash)
	subDirectory, filename := hash[:2], hash[2:]
	path := filepath.Join(GIT_OBJECTS_DIRECTORY, subDirectory, filename)

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "file not found")
		os.Exit(1)
	} else if err != nil {
		fmt.Fprint(os.Stderr, "issue with opening object file\n")
		os.Exit(1)
	}
	defer file.Close()

	// using the zlib read to decompress the object's content
	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decompressing object: %s\n", err)
		os.Exit(1)
	}
	defer zlibReader.Close()

	data, err := io.ReadAll(zlibReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading in content: %s\n", err)
		os.Exit(1)
	}

	// getting position of null byte "\0". it is actually just a 0 byte value. the "\" is just the escape for us to see
	// ex. blob 11\0hello world
	// splitting the bytes on the first null byte found so it because header bytes, content bytes, and found
	_, contentBytes, nullByteFound := bytes.Cut(data, []byte{0})
	if !nullByteFound {
		fmt.Fprintf(os.Stderr, "invalid object: no null byte after header\n")
		os.Exit(1)
	}

	content := string(contentBytes)
	fmt.Fprint(os.Stdout, content)
}
