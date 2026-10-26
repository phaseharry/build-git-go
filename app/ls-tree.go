package main

import (
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
	object, err := getObject(hash)
	if err != nil {
		os.Exit(1)
	}

	output := []byte{}
	objectReferences, err := getTreeReferences(object)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	for _, ref := range objectReferences {
		if *nameOnly {
			output = fmt.Appendf(output, "%s\n", ref.Name)
			continue
		}
		output = fmt.Appendf(output, string(ref.RawBytes))
	}

	fmt.Fprint(os.Stdout, string(output))
}
