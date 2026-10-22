package main

import (
	"fmt"
	"os"
)

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	command := os.Args[1]
	var args []string
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	switch command {
	case "init":
		gInit()
	case "cat-file":
		catFile(args)
	case "hash-object":
		hashObject(args)
	case "ls-tree":
		lsTree(args)
	case "write-tree":
		writeTree(args)
	case "commit-tree":
		commitTree(args)
	case "clone":
		clone(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
	os.Exit(0)
}
