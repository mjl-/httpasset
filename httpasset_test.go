package httpasset

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestHttpasset(t *testing.T) {
	tcheck := func(err error, action string) {
		t.Helper()

		if err != nil {
			t.Fatalf("%s: %s", action, err)
		}
	}

	f, err := binself()
	tcheck(err, "opening running binary")

	// Opening zip file in test binary should fail, it doesn't have a zip file yet.
	Fs()
	if Error() == nil {
		t.Fatalf("opening httpasset in test binary: got success, expected error")
	}
	_, err = fs.Open("/test.txt")
	if err == nil {
		t.Fatalf("failed httpasset unexpectedly succeeded opening a file")
	}

	// Let's append a zip file to the binary under test.
	// We have to make a copy because on some systems (ubuntu 16 lts is one) you cannot write to a running binary.
	newName := f.Name() + ".more"
	nf, err := os.Create(newName)
	tcheck(err, "creating new binary with zip")
	defer os.Remove(newName)
	_, err = io.Copy(nf, f)
	tcheck(err, "copy binary")
	f.Close()

	w := zip.NewWriter(nf)
	writeFile := func(name, contents string, compress bool) {
		t.Helper()

		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Store,
		}
		if compress {
			header.Method = zip.Deflate
		}
		ww, err := w.CreateHeader(header)
		tcheck(err, "adding zip file")
		_, err = ww.Write([]byte(contents))
		tcheck(err, "writing to file in zip")
	}
	writeFile("test.txt", "hi", false)
	writeFile("a/file1", "a", false)
	writeFile("a/compressed.txt", "compressed file", true)
	writeFile("b/c/d/e.txt", "e", false)
	err = w.Close()
	tcheck(err, "closing zip file writer")
	nf.Close()

	// Retry now that we added a zip file to the binary under test.
	os.Args[0] = newName
	Close()
	Fs()
	err = Error()
	tcheck(err, "opening zip file appended to test binary")

	// Paths must always be absolute.
	_, err = fs.Open("test.txt")
	if err != os.ErrNotExist {
		t.Fatalf("opening relative path: got success, expected os.ErrNotExist")
	}

	_, err = fs.Open("/bogus.txt")
	if err != os.ErrNotExist {
		t.Fatalf("opening bogus path: got success, expected os.ErrNotExist")
	}

	verifyFile := func(zf http.File, name, contents string, compressed bool) {
		t.Helper()

		buf, err := ioutil.ReadAll(zf)
		tcheck(err, "reading file from zip")
		if string(buf) != contents {
			t.Fatalf("bad content reading file from zip: got %#v, expected %#v\n", string(buf), contents)
		}

		off, err := zf.Seek(1, 0)
		if compressed {
			if err != errCompressedSeek {
				t.Fatalf("seek on compressed file: got %v, expected errCompressedSeek", err)
			}
		} else {
			tcheck(err, "seek on uncompressed file")
			if off != 1 {
				t.Fatalf("offset after seek: got %d, expected 1", off)
			}
		}

		st, err := zf.Stat()
		tcheck(err, "stat on file")
		if st.IsDir() || st.Name() != name {
			t.Fatalf("stat file, got %v %v", st.IsDir(), st.Name())
		}
		_, err = zf.Readdir(1)
		if err != ErrNotDir {
			t.Fatalf("readdir on file, got %v expected ErrNotDir", err)
		}
	}

	// Test an uncompressed file.
	zf, err := fs.Open("/test.txt")
	tcheck(err, "open file in included zip")
	verifyFile(zf, "test.txt", "hi", false)
	err = zf.Close()
	tcheck(err, "closing file from zip")

	// Test a ompressed file.
	zf, err = fs.Open("/a/compressed.txt")
	tcheck(err, "open file in included zip")
	verifyFile(zf, "compressed.txt", "compressed file", true)
	err = zf.Close()
	tcheck(err, "closing file from zip")

	// Test a directory.
	dir, err := fs.Open("/a")
	tcheck(err, "open directory")
	st, err := dir.Stat()
	tcheck(err, "stat on dir")
	if st != zerofileinfo {
		t.Fatalf("stat on dir, got %#v, expected zerofileinfo", st)
	}
	// Let's get these called.
	st.Name()
	st.Size()
	st.Mode()
	st.ModTime()
	st.IsDir()
	st.Sys()

	_, err = dir.Read(make([]byte, 1))
	if err != errReadOnDir {
		t.Fatalf("reading on dir: got %v, expected errReadOnDir", err)
	}
	_, err = dir.Seek(0, 0)
	if err != errSeekOnDir {
		t.Fatalf("seek on dir: got %v, expected errSeekOnDir", err)
	}
	l, err := dir.Readdir(1)
	tcheck(err, "readdir on dir")
	if len(l) != 0 {
		t.Fatalf("readdir on dir, got %v, expected zero entries", l)
	}

	err = dir.Close()
	tcheck(err, "closing dir")

	Close()
}
