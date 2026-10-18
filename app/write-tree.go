package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// no need for args at the moment
func writeTree(_ []string) {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting working directory: %s\n", err.Error())
		os.Exit(1)
	}
	treeHash := recursivelyWriteTree(workingDir)
	fmt.Fprintf(os.Stdout, "%x\n", treeHash)
}

func recursivelyWriteTree(directory string) []byte {
	type TreeEntry struct {
		Mode string
		Name string
		Hash []byte
	}

	// a sample leaf node where there is no subdirectory within it. ie only contains files
	// tree <size>\0
	// <mode> <name>\0<20_byte_sha>
	// <mode> <name>\0<20_byte_sha>
	entries, err := os.ReadDir(directory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening files in directory: %s\n", directory)
		os.Exit(1)
	}

	currentTreeEntries := make([]TreeEntry, 0)

	for _, e := range entries {
		path := filepath.Join(directory, e.Name())
		mode := REGULAR_FILE_MODE
		// checking to ensure we do not include .git files and subdirectories within the hashed results
		if e.IsDir() && e.Name() != ".git" {
			mode = DIRECTORY_MODE
			hash := recursivelyWriteTree(path)
			currentTreeEntries = append(currentTreeEntries, TreeEntry{
				Mode: mode,
				Name: e.Name(),
				Hash: hash,
			})
			continue
		} else if e.IsDir() && e.Name() == ".git" {
			continue
		}

		var contentBytes []byte
		// comparing e.Type() with a AND bitwise operator to see if the symlink flag is set
		// if it's a symlink, use the link to get the actual file content that it points to and load that into contentBytes
		if e.Type()&os.ModeSymlink != 0 {
			// a symlink's blob holds the path it points to, not the target's content
			mode = SYMBOLIC_LINK_MODE
			target, err := os.Readlink(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading symlink %s, err: %s\n", path, err.Error())
				os.Exit(1)
			}
			contentBytes = []byte(target)
		} else {
			// Type() only has the type bits, Info() is needed for the permission bits
			info, err := e.Info()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error getting file info %s, err: %s\n", path, err.Error())
				os.Exit(1)
			}
			// git only checks the owner's execute bit
			if info.Mode().Perm()&0100 != 0 {
				mode = EXECUTABLE_FILE_MODE
			}

			contentBytes, err = os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error opening file %s, err: %s\n", path, err.Error())
				os.Exit(1)
			}
		}

		data, hash := encodeObject(BLOB, contentBytes)
		if err := writeObject(hash, data); err != nil {
			os.Exit(1)
		}
		currentTreeEntries = append(currentTreeEntries, TreeEntry{
			Mode: mode,
			Name: e.Name(),
			Hash: hash,
		})
	}

	treeBytes := make([]byte, 0)
	for _, e := range currentTreeEntries {
		treeBytes = append(treeBytes, []byte(e.Mode)...)
		treeBytes = append(treeBytes, ' ')
		treeBytes = append(treeBytes, []byte(e.Name)...)
		treeBytes = append(treeBytes, 0)
		// tree entries store the raw 20-byte hash, not the 40-char hex string
		treeBytes = append(treeBytes, e.Hash...)
	}
	data, treeHash := encodeObject(TREE, treeBytes)
	if err := writeObject(treeHash, data); err != nil {
		fmt.Fprintf(os.Stderr, "error writing tree object: %s", err.Error())
		os.Exit(1)
	}

	return treeHash
}
