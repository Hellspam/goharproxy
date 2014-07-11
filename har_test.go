package goharproxy

import (
	"testing"
	"net/http"
	"bytes"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

func TestParseHttpGETRequest (t *testing.T) {
	req, _ := http.NewRequest("GET", "http://google.com", nil)
	if req == nil {
		t.Errorf("Failure creating request")
	}

	expectedReq := HarRequest{
		Method 		: "GET",
		Url    		: "http://google.com",
		BodySize 	: 0,
	}

	if harReq := parseRequest(req); reflect.DeepEqual(expectedReq, harReq) {
		t.Errorf("Expected:\n %v \n\n Actual:\n %v \n\n", expectedReq, harReq)
	}
}

func TestParseIpString (t *testing.T) {

	req, _ := http.NewRequest("GET", "http://google.com", nil)
	if req == nil {
		t.Errorf("Failure creating request")
	}

	expectedReq := HarRequest{
		Method 		: "GET",
		Url    		: "http://google.com",
		BodySize 	: 0,
	}

	if harReq := parseRequest(req); reflect.DeepEqual(expectedReq, harReq) {
		t.Errorf("Expected:\n %v \n\n Actual:\n %v \n\n", expectedReq, harReq)
	}
}

func TestParseHttpIpGETRequest (t *testing.T) {
	req, _ := http.NewRequest("GET", "http://google.com", nil)
	if req == nil {
		t.Errorf("Failure creating request")
	}

	expectedReq := HarRequest{
		Method 		: "GET",
		Url    		: "http://google.com",
		BodySize 	: 0,
	}

	if harReq := parseRequest(req); reflect.DeepEqual(expectedReq, harReq) {
		t.Errorf("Expected:\n %v \n\n Actual:\n %v \n\n", expectedReq, harReq)
	}
}

func TestParseHttpPOSTRequest (t *testing.T) {
	req, expectedReq := getTestSendRequest("POST", t)
	captureContent = true
	if harReq := parseRequest(req); reflect.DeepEqual(expectedReq, harReq) {
		t.Errorf("Expected:\n %v \n\n Actual:\n %v \n\n", expectedReq, harReq)
	}
}

func TestParseHttpPUTRequest (t *testing.T) {
	req, expectedReq := getTestSendRequest("PUT", t)
	captureContent = true
	if harReq := parseRequest(req); reflect.DeepEqual(expectedReq, harReq) {
		t.Errorf("Expected:\n %v \n\n Actual:\n %v \n\n", expectedReq, harReq)
	}
}

func TestParsePostDataRaw(t *testing.T) {
	testString := "BLA"
	req, err := http.NewRequest("POST", "http://google.com", strings.NewReader(testString))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "Raw")
	contentLength := strconv.Itoa(len(testString))
	req.Header.Add("Content-Length", contentLength)
	postData := parsePostData(req)
	if postData.Text != testString {
		t.Fatal("Did not get expected text")
	}
}

func getTestSendRequest(method string, t *testing.T) (*http.Request, *HarRequest) {
	data := url.Values{}
	data.Set("name", "foo")
	data.Add("surname", "bar")
	req, err := http.NewRequest(method, "http://google.com", bytes.NewBufferString(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	contentType := "application/x-www-form-urlencoded"
	req.Header.Add("Content-Type", contentType)
	contentLength := strconv.Itoa(len(data.Encode()))
	req.Header.Add("Content-Length", contentLength)

	index  := 0
	params := make([]HarPostDataParam, len(data))
	for k, v := range data {
		param := HarPostDataParam {
			Name  : k,
			Value : strings.Join(v, ","),
		}
		params[index] = param
		index++
	}

	harPostData := HarPostData {
		Params 	   : params,
		MimeType   : contentType,
	}
	expectedReq := HarRequest{
		Method 		: method,
		Url    		: "http://google.com",
		BodySize 	: (int64)(len(data.Encode())),
		PostData	: &harPostData,
	}

	return req, &expectedReq
}


