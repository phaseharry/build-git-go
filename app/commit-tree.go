package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"os"
)

func commitTree(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: commit-tree <tree_sha>")
		os.Exit(1)
	}
	// have to pull out the required input hash first because fs.Parse stops parsing after the first non flag argument
	// that's why we strip out the treeHash in the slice before we send it to parse
	treeHash := args[0]

	fs := flag.NewFlagSet("commit-tree", flag.ExitOnError)
	parent := fs.String("p", "", "parent hash")
	message := fs.String("m", "", "commit message")
	fs.Parse(args[1:])

	// commit 177\0tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904
	// parent 3b18e512dba79e4c8300dd08aeb37f8e728b8dad
	// author John Doe <john@example.com> 1234567890 +0000
	// committer John Doe <john@example.com> 1234567890 +0000
	//
	// Initial commit
	content := make([]byte, 0)
	content = append(content, fmt.Appendf(nil, "tree %s\n", treeHash)...)
	if *parent != "" {
		content = append(content, fmt.Appendf(nil, "parent %s\n", *parent)...)
	}
	// hardcoding author, committer, and timestamp
	name, email := "John Doe", "john@example.com"
	secondsSinceEpoch, timezoneOffset := "1234567890", "+0000"

	content = append(content,
		fmt.Appendf(nil, "author %s <%s> %s %s\n", name, email, secondsSinceEpoch, timezoneOffset)...)
	content = append(content,
		fmt.Appendf(nil, "committer %s <%s> %s %s\n", name, email, secondsSinceEpoch, timezoneOffset)...)

	if *message != "" {
		content = append(content, '\n')
		content = append(content, []byte(*message)...)
	}

	// need to attach all content bytes first so we can get the size of it
	data := make([]byte, 0)
	data = append(data, fmt.Appendf(nil, "commit %d\x00", len(content))...)
	// can't reuse existing encodeObject function because it the header line contains an extra hash
	h := sha1.New()
	h.Write(data)
	commitHash := h.Sum(nil)
	writeObject(commitHash, data)
	fmt.Fprintf(os.Stdout, "%x\n", commitHash)
}
