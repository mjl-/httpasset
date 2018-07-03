// Httpassetexample show how to use the httpasset library.
// It serves files over http at :8000. If the binary had a zip-file appended, it serves files from the root of the zip file. Otherwise it serves files from the ./assets/ in the local file system.

package main

import (
	"log"
	"net/http"
	"bitbucket.org/mjl/httpasset"
)

var httpFS http.FileSystem

func init() {
	// the error-check is optional, httpasset.Fs() always returns a non-nil http.FileSystem.
	// however, after failed initialization (eg no zip file was appended to the binary),
	// fs operations return an error.
	httpFS = httpasset.Fs()
	if err := httpasset.Error(); err != nil {
		// note: could also just quit here
		log.Print("falling back to local assets")
		httpFS = http.Dir("assets")
	}
}

func main() {
	http.Handle("/", http.FileServer(httpFS))
	addr := ":8000"
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
