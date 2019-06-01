// Httpassetexample shows how to use the httpasset library.
//
// It serves files over http at :8000. If the binary had a zip-file appended, it
// serves files from the root of the zip file. Otherwise it serves files from
// ./assets/ in the local file system.

package main

import (
	"log"
	"net/http"

	"bitbucket.org/mjl/httpasset"
)

var httpFS = httpasset.Init("assets")

func main() {
	http.Handle("/", http.FileServer(httpFS))
	addr := ":8000"
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
