package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
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
	data, err := getObject(hash)
	if err != nil {
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
