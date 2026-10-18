package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"os"
)

func lsTree(args []string) {
	fs := flag.NewFlagSet("ls-tree", flag.ExitOnError)
	nameOnly := fs.Bool("name-only", false, "print name-only of the objects within the tree")
	fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: mygit ls-tree <object>\n")
		os.Exit(1)
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
	remaining := contentBytes
	// every iteration parses out each content entry
	for len(remaining) > 0 {
		// <mode> ends at the first space
		mode, afterMode, found := bytes.Cut(remaining, []byte{' '})
		if !found {
			fmt.Fprintf(os.Stderr, "invalid tree entry: no null byte after name\n")
			os.Exit(1)
		}

		// <name> ends at the null byte
		name, afterName, found := bytes.Cut(afterMode, []byte{0})
		if !found {
			fmt.Fprintf(os.Stderr, "invalid tree entry: no null byte after name\n")
			os.Exit(1)
		}

		// <20_byte_sha> has no terminator.
		// it's always exactly 20 raw bytes and the next object entry starts right after it
		if len(afterName) < sha1.Size {
			fmt.Fprintf(os.Stderr, "invalid tree entry: sha is shorter than %d bytes\n", sha1.Size)
			os.Exit(1)
		}

		// getting sha bytes and update remaining byte slice to strip it of the current entry we've parsed
		sha := afterName[:sha1.Size]
		remaining = afterName[sha1.Size:]

		// if name-only flag is set then only print that and skip the rest
		if *nameOnly {
			output = fmt.Appendf(output, "%s\n", name)
			continue
		}

		// only directories are trees. regular files, executables, and symlinks are all blobs
		objectType := "blob"
		if string(mode) == DIRECTORY_MODE {
			objectType = "tree"
		}

		// full entry output
		// 040000 tree <tree_sha_1>    dir1
		// 100644 blob <blob_sha_1>    file1
		// %06s prints the mode bytes as text and zero-pads "40000" to "040000" like git does, %x turns the raw sha bytes into hex
		output = fmt.Appendf(output, "%06s %s %x\t%s\n", mode, objectType, sha, name)
	}
	fmt.Fprint(os.Stdout, string(output))
}
