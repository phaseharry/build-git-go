package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
)

type ObjectReference struct {
	Name       string
	Mode       string
	SHA        string
	ObjectType string
	RawBytes   []byte
}

// tree object format
// tree <size>\0
// <mode> <name>\0<20_byte_sha>
// <mode> <name>\0<20_byte_sha>
func getTreeReferences(object Object) ([]ObjectReference, error) {
	objectReferences := []ObjectReference{}

	remaining := object.Content
	// every iteration parses out each content entry
	for len(remaining) > 0 {
		// <mode> ends at the first space
		mode, afterMode, found := bytes.Cut(remaining, []byte{' '})
		if !found {
			return objectReferences, fmt.Errorf("invalid tree entry: no null byte after name\n")
		}

		// <name> ends at the null byte
		name, afterName, found := bytes.Cut(afterMode, []byte{0})
		if !found {
			return objectReferences, fmt.Errorf("invalid tree entry: no null byte after name\n")
		}

		// <20_byte_sha> has no terminator.
		// it's always exactly 20 raw bytes and the next object entry starts right after it
		if len(afterName) < sha1.Size {
			return objectReferences, fmt.Errorf("invalid tree entry: sha is shorter than %d bytes\n")
		}

		// getting SHA bytes and update remaining byte slice to strip it of the current entry we've parsed
		SHA := afterName[:sha1.Size]
		remaining = afterName[sha1.Size:]

		// only directories are trees. regular files, executables, and symlinks are all blobs
		objectType := "blob"
		if string(mode) == DIRECTORY_MODE {
			objectType = "tree"
		}

		// full entry output
		// 040000 tree <tree_sha_1>    dir1
		// 100644 blob <blob_sha_1>    file1
		// %06s prints the mode bytes as text and zero-pads "40000" to "040000" like git does, %x turns the raw sha bytes into hex
		rawBytes := []byte(fmt.Sprintf("%06s %s %x\t%s\n", mode, objectType, SHA, name))

		objectReferences = append(objectReferences, ObjectReference{
			Name:       string(name),
			Mode:       string(mode),
			SHA:        string(SHA),
			ObjectType: objectType,
			RawBytes:   rawBytes,
		})
	}

	return objectReferences, nil
}
