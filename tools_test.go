package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("Random string length is not 10")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png", "image/gif"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png", "image/gif"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg", "image/gif"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// set up a pipe to avoid buffering

		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// CREATE A FORM DATA FIELD FILE
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Errorf("Error creating form file: %v", err)
			}
			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Errorf("Error opening file: %v", err)
			}
			defer f.Close()
			img, _, err := image.Decode(f)
			if err != nil {
				t.Errorf("Error decoding image: %v", err)
			}
			err = png.Encode(part, img)
			if err != nil {
				t.Errorf("Error encoding image: %v", err)
			}
		}()

		// read from the pipe which receives the data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Errorf("Error uploading file: %v", err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: File not uploaded: %s ", e.name, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: Error uploading file: %v, error expected but none received", e.name, err)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {

	// set up a pipe to avoid buffering

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// CREATE A FORM DATA FIELD FILE
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Errorf("Error creating form file: %v", err)
		}
		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Errorf("Error opening file: %v", err)
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			t.Errorf("Error decoding image: %v", err)
		}
		err = png.Encode(part, img)
		if err != nil {
			t.Errorf("Error encoding image: %v", err)
		}
	}()

	// read from the pipe which receives the data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadFile, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Errorf("Error uploading file: %v", err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("File not uploaded: %s ", err.Error())
	}

	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadFile.NewFileName))

}

func TestTools_CreateDirIfNotExists(t *testing.T) {
	var testTools Tools

	err := testTools.CreateDirIfNotExists("./testdata/myDir")
	if err != nil {
		t.Errorf("Error creating directory: %v", err)
	}

	err = testTools.CreateDirIfNotExists("./testdata/myDir")
	if err != nil {
		t.Errorf("Error creating directory: %v", err)
	}
	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "This is a valid string", expected: "this-is-a-valid-string", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "string with numbers", s: "This is a valid string 123", expected: "this-is-a-valid-string-123", errorExpected: false},
	{name: "string with characters", s: "Th*İSs *eĞcspeCtDe!", expected: "th-iss-e-cspectde", errorExpected: false},
	{name: "japanese string", s: "これは日本語の文字列です", expected: "", errorExpected: true},
	{name: "japanese chars with roman characters", s: "これは日本語の文字列ですhello world", expected: "hello-world", errorExpected: true},
}

func TestTools_Slugify(t *testing.T) {
	for _, e := range slugTests {
		var testTools Tools

		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: Error slugifying string: %v", e.name, err)
		}

		if slug != e.expected {
			t.Errorf("%s: Slug not as expected: %s", e.name, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, req, "./testdata", "foto.png", "foti.png")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "1742391" {
		t.Errorf("Content-Length not as expected: %s", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"foti.png\"" {
		t.Errorf("Content-Disposition not as expected: %s", res.Header["Content-Disposition"][0])
	}

	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("Error reading response body: %v", err)
	}

}
