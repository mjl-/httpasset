package httpasset

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

// Disabled until os.Executable can be mocked. Not sure if we want to go there.
func disabledTestHttpasset(t *testing.T) {
	tcheck := func(err error, action string) {
		t.Helper()

		if err != nil {
			t.Fatalf("%s: %s", action, err)
		}
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
		if err == nil {
			t.Fatalf("got nil for readdir on file, expected error")
		}
	}

	name, err := os.Executable()
	tcheck(err, "finding running binary")
	f, err := os.Open(name)
	tcheck(err, "opening running binary")

	// Opening zip file in test binary should fail, it doesn't have a zip file yet.
	_, err = ZipFS()
	if err == nil {
		t.Fatalf("opening httpasset in test binary: got success, expected error")
	}

	// Test with fallback.
	fs := Init("testdata")
	_, err = fs.Open("hi.txt")
	if err != os.ErrNotExist {
		t.Fatalf("got err %v for open of file not starting with /, expected os.ErrNotExist", err)
	}
	zf, err := fs.Open("/hi.txt")
	tcheck(err, "open file from fallback dir")
	verifyFile(zf, "hi.txt", "hi\n", false)
	err = zf.Close()
	tcheck(err, "closing file from zip")
	fs.Close()

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
	fs, err = ZipFS()
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

	// Test an uncompressed file.
	zf, err = fs.Open("/test.txt")
	tcheck(err, "open file in included zip")
	verifyFile(zf, "test.txt", "hi", false)
	err = zf.Close()
	tcheck(err, "closing file from zip")

	// Test a compressed file.
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

	fs.Close()
	_, err = fs.Open("/test.txt")
	if err != os.ErrClosed {
		t.Fatalf("got %v for open on closed fs, expected os.ErrClosed", err)
	}
	fs.Close()

	fs = Init("testdata")
	zf, err = fs.Open("/test.txt")
	tcheck(err, "open file in included zip")
	verifyFile(zf, "test.txt", "hi", false)
	err = zf.Close()
	tcheck(err, "closing file from zip")
	fs.Close()
}
