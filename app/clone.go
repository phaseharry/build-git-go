package main

import (
	"bytes"
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

	getGitUploadPack(args[0], headSHA)
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
func getGitUploadPack(repoLink string, targetSHA []byte) {
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
		fmt.Fprintf(os.Stderr, "error fetching for git-upload-pack: %s\n", err.Error())
		os.Exit(1)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	fmt.Printf("%q", body[:12])
}
