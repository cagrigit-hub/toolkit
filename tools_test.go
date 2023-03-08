package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
