package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

func clone(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: clone <repo-link>")
		os.Exit(1)
	}

	lines, err := getGitPackUploadInfo(args[0])
	if err != nil {
		os.Exit(1)
	}

	headSHA, err := getHeadSHA(lines)
	if err != nil {
		os.Exit(1)
	}

	body, err := getGitUploadPackResponse(args[0], headSHA)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while getting the git pack response: %s\n", err.Error())
		os.Exit(1)
	}

	validResponse := validateResponseChecksum(body)
	if !validResponse {
		fmt.Fprint(os.Stderr, "git pack response's checksum did not match the computed one\n")
		os.Exit(1)
	}

	err = processObjectEntries(body)
}

func getGitPackUploadInfo(repoLink string) ([][]byte, error) {
	// ex. https://github.com/phaseharry/build-http-server-go.git/info/refs?service=git-upload-pack
	// need to call this ahead of time to see what state of git is on the server's end
	res, err := http.Get(fmt.Sprintf("%s.git/info/refs?service=git-upload-pack", repoLink))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching for git-upload-pack info: %s\n", err.Error())
		return nil, err
	}
	if res.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "git repo: %s is not found\n", repoLink)
		return nil, fmt.Errorf("git repo not found")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading response body: %s", err.Error())
		return nil, err
	}

	lines, err := parsePktLines(body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing pkt lines: %s", err.Error())
		return nil, err
	}

	return lines, nil
}

// consumes the raw body response of git-upload-pack and parses it into lines.
// each line follows the same structure
// <4 bytes> to indicate the size of the line & the rest for the line's content. the space those 4 bytes take up is included as part of the size
// 0000 being the exception of being nothing and just an empty line
//
// 001e  # service=git-upload-pack\n              header (0x1e = 30 bytes, including "001e")
// 0000                                           flush-pkt: ends the header section
// 0159  2a6b…4ba3 HEAD\0multi_ack thin-pack …\n  first ref; capabilities come after a NUL byte
// 003d  2a6b…4ba3 refs/heads/main\n
// 003f  2a6b…4ba3 refs/heads/master\n
// 0000                                           flush-pkt: ends the ref list
func parsePktLines(data []byte) ([][]byte, error) {
	lines := [][]byte{}
	remaining := data

	for len(remaining) > 0 {
		if len(remaining) < 4 {
			return nil, fmt.Errorf("pkt-line is missing its 4 byte length prefix")
		}
		length, err := strconv.ParseUint(string(remaining[:4]), 16, 16) // parsing hex digits so need base 16 and bit 16
		if err != nil {
			return nil, fmt.Errorf("error parsing line length: %s", err.Error())
		}

		// 0000 takes up 4 bytes but there's nothing after it
		if length == 0 {
			remaining = remaining[4:]
			continue
		}

		if length < 4 || int(length) > len(remaining) {
			return nil, fmt.Errorf("invalid pkt-line length %d", length)
		}

		lines = append(lines, remaining[4:length])
		remaining = remaining[length:]
	}
	return lines, nil
}

// takes the parsed PktLines and pull the repo's HEAD SHA
// here's an example of the lines.
// # service=git-upload-pack
// f56c5754e0a6555fadf3ecb5538225ecab9c4d6d HEADmulti_ack thin-pack side-band side-band-64k ofs-delta shallow deepen-since deepen-not deepen-relative no-progress include-tag multi_ack_detailed allow-tip-sha1-in-want allow-reachable-sha1-in-want no-done symref=HEAD:refs/heads/main filter object-format=sha1 agent=git/github-f6e509caf74d-Linux
// f56c5754e0a6555fadf3ecb5538225ecab9c4d6d refs/heads/main
// f56c5754e0a6555fadf3ecb5538225ecab9c4d6d refs/heads/master
func getHeadSHA(lines [][]byte) ([]byte, error) {
	for _, line := range lines {
		fmt.Print(string(line))
	}
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid git-upload-pack info: not enough lines")
	}
	headerLine := lines[1]
	headSHA, _, _ := bytes.Cut(headerLine, []byte(" "))
	return headSHA, nil
}

// used to get the objects reachable by the SHA / commit you want.
// since we first called the getPackInfo to get the HEAD SHA, this will give us every object reachable by that commit.
// this includes every commit, trees, and blob in its history
func getGitUploadPackResponse(repoLink string, targetSHA []byte) ([]byte, error) {
	reqBody := []byte{}
	// %04x ensures the input value will be converted to a 4 bytes
	// 0032want f56c5754e0a6555fadf3ecb5538225ecab9c4d6d\n
	// 4 bytes for the size, 4 bytes for "want", 1 byte for the <space>, 1 byte for the '\n', and the length of the SHA1
	reqBody = fmt.Appendf(reqBody, "%04xwant %s\n", len(targetSHA)+4+4+1+1, targetSHA)
	// 0000 <- size of 0 to indicate seperate between header and the done signal
	reqBody = fmt.Appendf(reqBody, "%04x", 0)
	// 0009done\n <- 9 bytes including the size byes
	reqBody = fmt.Appendf(reqBody, "%04x%s\n", 9, "done")
	updateLoadPack := ".git/git-upload-pack"

	res, err := http.Post(
		fmt.Sprintf("%s%s", repoLink, updateLoadPack),
		"application/x-git-upload-pack-request",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error fetching for git-upload-pack: %s\n", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-OK message. statusCode: %d", res.StatusCode)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func validateResponseChecksum(body []byte) bool {
	// the last 20 bytes of the body should be the SHA checksum for the response body
	// it is generated on text after the first 8 bytes (0008NAK) and before the last 20 bytes
	// as that is where the SHA is stored within the response
	computedSHA := sha1.Sum(body[8 : len(body)-20])
	receivedSHA := body[len(body)-20:]
	return bytes.Equal(computedSHA[:], receivedSHA)
}

func processObjectEntries(body []byte) error {

	// 0008NAK\n           bytes 0–7     pkt-line: NAK
	// PACK                bytes 8–11    pack signature
	// \x00\x00\x00\x02    bytes 12–15   version 2
	// \x00\x00\x00j       bytes 16–19   object count: 106
	// \x97*x\x9c…         byte 20 on    the first object
	packHeader := body[:20]

	// indicates the status of the pack requests
	// since this is a fresh clone, we expect 0008NAK\n
	// prefix is the size of the message including the size byes & the '\n' and NAK.
	// this means there are no objects in common as expected since it's a fresh clone
	packStatus := string(packHeader[4:7])
	if packStatus != "NAK" {
		return fmt.Errorf("want pack status to be NAK as its an initial clone, got %s\n", packStatus)
	}

	// calling BigEndian because that's the format that's sent
	objectCount := binary.BigEndian.Uint32(packHeader[16:20]) // reads the first 4 bytes of input since it's 32bits
	fmt.Println(objectCount)
	reader := bytes.NewReader(body[20:])

	for range objectCount {
		// reading the object's header byte to get details on its type and its size
		b, _ := reader.ReadByte()
		// gets the header byte (0x97 = 1001 0111) and shifts it 4 bits to the left
		// 1001 0111 -> 0000 1001 (keeping the leftmost 4 bits)
		// then runs a bitwise AND with 111 to keep only the lowest 3 bits after we shifted which is used to define the object type. (001) == commit && (111) == ref-delta
		// objectType == bit [6,4]
		objectTypeVal := (b >> 4) & 0b111
		objectType, err := getObjectTypeByValue(int(objectTypeVal))
		if err != nil {
			return err
		}

		// b & 0b1111 does a bitwise & with only the rightmost 4 bits of the byte.
		// that's the start of the size
		// size = [3,0]
		size := int(b & 0b1111)
		shift := 4 // the header's first size byte will always have a fixed 4 bits allocated to the size of the objects uncompressed.

		// leftmost bit (bit 7) is a flag indicating if there's another message to this byte.
		// keep iterating bytes until we see a byte without bit 7 position not set to 1.
		// that is the last bit
		// checking if the 7th bit is set to 1. this can be done with a more readable int 128 or hex 0x80.
		// that's because 128 is the int value if only that bit is set
		// 0b10000000 == 128 == 0x80
		for b&0b10000000 != 0 {
			b, err = reader.ReadByte()
			if err != nil {
				return fmt.Errorf("error reading ref-delta SHA byte: %s\n", err.Error())
			}

			// comparing the current byte to bit positions [6:0] as that's where the additional size bits are.
			// 0b1111111 == 127 == 0x7f
			// using bitwise AND to get the actual value by masking it
			additionalSizeBits := b & 0b1111111
			// converting it to an int so there's more available size to shift it leftward.
			// if left as a byte, we would be losing the leftmost bits as it overfills out the 8 slots
			// using a bitwise OR to add the results of the previous size byte(s) to current size byte
			//
			// after each
			size |= int(additionalSizeBits) << shift
			shift += 7
		}
		// see example of calculate size bytes & the shift that's needed
		//	byte 1  0xB5 = 1 011 0101  → more follows, type 3 (blob), size bits 0101
		//	byte 2  0x8C = 1 000 1100  → more follows, 12
		//	byte 3  0xA3 = 1 010 0011  → more follows, 35
		//	byte 4  0x02 = 0 000 0010  → last byte, 2
		//
		//						5      =                            0101
		// 12 << 4  = 192    =                   000 1100 0000
		// 35 << 11 = 71680  =          010 0011 000 0000 0000
		//  2 << 18 = 524288 = 000 0010 000 0000 000 0000 0000
		// OR       = 596165 = 000 0010 010 0011 000 1100 0101 (total size)

		// ref-delta has a 20 byte SHA before the actual zlip compressed content.
		// so to not panic or error out on an invalid gzip or incorrectly skipping to next object and malform our data,
		// explicitly skip 20 SHA bytes by moving the reader pointer 20 bytes forward
		// the actual ref-delta content will still be decompressed, encoded, and written properly
		if objectType == REF_DELTA {
			for range 20 {
				_, err := reader.ReadByte()
				if err != nil {
					return fmt.Errorf("error reading ref-delta SHA byte: %s\n", err.Error())
				}
			}
		}

		// can reuse the reader as each time a byte is read, the reader pointer shifts to the next byte to be read
		// zlip knows exactly what bytes are compressed and will shift the reader pointer until the end of the compressed data and decompress.
		//
		// ex. gzip content
		// 78 9c                 2-byte zlib header
		// [deflate blocks]      the compressed data; the last block has a "final" flag
		// [4-byte checksum]     Adler-32 of the decompressed content
		//
		// this also shifts our reader pointer for us to we can get the next object entry
		zr, err := zlib.NewReader(reader)

		content, err := io.ReadAll(zr)
		zr.Close()

		data, hash := encodeObject(objectType, content)
		err = writeObject(hash, data)
		fmt.Printf("%x\n", hash)
	}

	return nil
}
