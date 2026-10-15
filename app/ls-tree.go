package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
)

const REGULAR_FILE_MODE = 100644
const EXECUTABLE_FILE_MODE = 100755
const SYMBOLIC_LINK_MODE = 120000
const DIRECTORY_MODE = 40000

func lsTree(args []string) {
	fs := flag.NewFlagSet("cat-file", flag.ExitOnError)
	nameOnly := fs.Bool("name-only", false, "print name-only of the objects within the tree")
	fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: mygit ls-tree <object>\n")
	}

	hash := fs.Arg(0)
	data, err := getObject(hash)
	if err != nil {
		os.Exit(1)
	}
	// tree object format
	// tree <size>\0
	// <mode> <name>\0<20_byte_sha>
	// <mode> <name>\0<20_byte_sha>

	// removing the header section to be able to get file & directory info
	_, contentBytes, nullByteFound := bytes.Cut(data, []byte{0})
	if !nullByteFound {
		fmt.Fprintf(os.Stderr, "invalid object: no null byte after header\n")
		os.Exit(1)
	}

	output := []byte{}

	// without --name-only flag
	// 040000 tree <tree_sha_1>    dir1
	// 040000 tree <tree_sha_2>    dir2
	// 100644 blob <blob_sha_1>    file1

	// with --name-only flag
	// dir1
	// dir2
	// file1
}
