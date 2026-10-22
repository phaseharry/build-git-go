package main

import (
	"fmt"
	"net/http"
	"os"
)

func clone(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: clone <repo-link>")
		os.Exit(1)
	}

	repoLink := args[0]
	// ex. https://github.com/phaseharry/build-http-server-go.git/info/refs?service=git-upload-pack
	// need to call this ahead of time to see what state of git is on the server's end
	response, err := http.Get(fmt.Sprintf("%s.git/info/refs?service=git-upload-pack", repoLink))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching for git-upload-pack: %s\n", err.Error())
		os.Exit(1)
	}
	if response.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "git repo: %s is not found\n", repoLink)
		os.Exit(1)
	}
	defer response.Body.Close()

}
