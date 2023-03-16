package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
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

// small change for test git

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONTORemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("OK")),
			Header:     make(http.Header),
		}
	})

	var testTools Tools
	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "foo"

	_, _, err := testTools.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Errorf("Error pushing JSON to remote: %v", err)
	}

}

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

	testTools.DownloadStaticFile(rr, req, "./testdata/foto.png", "foti.png")

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

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int64
	allowUnknown  bool
}{
	{name: "good json", json: `{"name": "John", "age": 30, "city": "New York"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "bad json", json: `{"name": "John", "age": 30, "city": "New York"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json with unknown fields", json: `{"name": "John", "age": 30, "city": "New York", "country": "USA"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json with unknown fields allowed", json: `{"name": "John", "age": 30, "city": "New York", "country": "USA"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "json with max size", json: `{"name": "John", "age": 30, "city": "New York", "country": "USA"}`, errorExpected: true, maxSize: 10, allowUnknown: false},
	{name: "incorrect type", json: `{"name": 30, "age": 30, "city": "New York"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json file", json: `{"name": "John", "age": 30, "city": "New York"}{"name": "John", "age": 30, "city": "New York"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "not json", json: `this is not json`, errorExpected: true, maxSize: 1024, allowUnknown: false},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools
	for _, e := range jsonTests {
		// set the max file size
		testTools.MaxJSONSize = e.maxSize

		// allow/disallow unknown fields
		testTools.AllowUnknownFields = e.allowUnknown

		// declare a var to read decoded json
		var decodedJSON struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
			City string `json:"city"`
		}

		// create a request with the body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		// create a recorder
		rr := httptest.NewRecorder()
		err = testTools.ReadJSON(rr, req, &decodedJSON)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: Error reading json: %v", e.name, err)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: Error reading json: %v", e.name, err)
		}

		req.Body.Close()

	}
}

func TestTools_WriteJson(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "test",
	}
	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, payload, http.StatusOK, headers)

	if err != nil {
		t.Errorf("Error writing json: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()

	err := testTools.ErrorJSON(rr, errors.New("Some error"), http.StatusInternalServerError)

	if err != nil {
		t.Errorf("Error writing json: %v", err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Errorf("Error decoding json: %v", err)
	}

	if !payload.Error {
		t.Errorf("Error field not set to true")
	}

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Status code not set to 500")
	}

}
