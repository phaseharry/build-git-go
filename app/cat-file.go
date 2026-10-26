package main

import (
	"flag"
	"fmt"
	"os"
)

func catFile(args []string) {
	fs := flag.NewFlagSet("cat-file", flag.ExitOnError)
	pretty := fs.Bool("p", false, "pretty-print object contents")
	fs.Parse(args)

	// checks the number of arguments that are not flags after all flag values have been parsed
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p <object>\n")
		os.Exit(1)
	}

	// flags are required for cat-file command. for now we only support -p
	if !*pretty {
		fmt.Fprint(os.Stderr, "usage: requires a flag to be set\n")
		os.Exit(1)
	}

	hash := fs.Arg(0)
	object, err := getObject(hash)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting object: %s", err.Error())
		os.Exit(1)
	}

	content := string(object.Content)
	fmt.Fprint(os.Stdout, content)
}
