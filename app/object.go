package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

const BLOB = "blob"
const TREE = "tree"
const COMMIT = "commit"
const TAG = "tag"
const REF_DELTA = "ref-delta"

const REGULAR_FILE_MODE = "100644"
const EXECUTABLE_FILE_MODE = "100755"
const SYMBOLIC_LINK_MODE = "120000"
const DIRECTORY_MODE = "40000"

// takes a file path and opens the open to get its raw bytes and then decompress it
func getObject(hash string) ([]byte, error) {
	subDirectory, filename := hash[:2], hash[2:]
	path := filepath.Join(GIT_OBJECTS_DIRECTORY, subDirectory, filename)

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "file not found")
		return []byte{}, err
	} else if err != nil {
		fmt.Fprint(os.Stderr, "issue with opening object file\n")
		return []byte{}, err
	}
	defer file.Close()

	// using the zlib read to decompress the object's content
	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decompressing object: %s\n", err)
		return []byte{}, err
	}
	defer zlibReader.Close()

	data, err := io.ReadAll(zlibReader)
	return data, err
}

func writeObject(hash []byte, data []byte) error {
	// object paths use the hex form of the hash: .git/objects/<first 2 chars>/<remaining 38 chars>
	hexHash := hex.EncodeToString(hash)
	directory := hexHash[:2]
	filename := hexHash[2:]

	// make directory if it doesn't already exist
	directoryPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory)
	if err := os.MkdirAll(directoryPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating object directory: %s, err: %s\n", directoryPath, err.Error())
		return err
	}

	// compressing writeBytes before it is written to file
	var buf bytes.Buffer
	zipWriter := zlib.NewWriter(&buf)
	if _, err := zipWriter.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "error compressing content %s\n", err.Error())
		return err
	}
	// calling Close flushes the rest of the bytes. buf isn't complete until Close() returns
	if err := zipWriter.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error compressing content %s\n", err.Error())
		return err
	}

	targetPath := filepath.Join(GIT_OBJECTS_DIRECTORY, directory, filename)
	if err := os.WriteFile(targetPath, buf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing object to file, err: %s\n", err.Error())
		return err
	}
	return nil
}

/*
 * takes incoming content bytes and convert it to the expected object format before it is compressed and stored.
 * also generates the sha1 hash for this object
 * returns (object bytes, raw 20-byte hash). use %x or hex.EncodeToString when the hash needs to be text
 */
func encodeObject(objectType string, content []byte) ([]byte, []byte) {
	size := strconv.Itoa(len(content))

	// blob 11\0hello world
	data := []byte{}
	data = append(data, []byte(objectType)...)
	data = append(data, ' ')
	data = append(data, []byte(size)...)
	data = append(data, 0) // appending null byte
	data = append(data, content...)

	// creates a new hasher that can take in bytes and when it has all the required bytes
	// call .Sum to get the hashed output
	h := sha1.New()
	h.Write(data)
	hashedBytes := h.Sum(nil)

	return data, hashedBytes
}

// this info is passed through the wire via 1 byte so there's not enough
// bytes to spell out the actual object type value. it is mapped to an int value
func getObjectTypeByValue(val int) (string, error) {
	switch val {
	case 1:
		return COMMIT, nil
	case 2:
		return TREE, nil
	case 3:
		return BLOB, nil
	case 4:
		return TAG, nil
	case 7:
		return REF_DELTA, nil
	default:
		return "", fmt.Errorf("unsupported object type: %s", val)
	}
}
