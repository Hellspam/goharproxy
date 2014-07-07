package har

import (
	"time"
	"encoding/json"
	"net/http"
	"strings"
)

type Har struct {
	HarLog HarLog
}

type HarLog struct {
	Pages   []HarPage
	Entries []HarEntry
}

type HarPage struct {
	Id              string
	StartedDateTime time.Time
	Title           string
	PageTimings     HarPageTimings
}

type HarEntry struct {
	PageRef         string
	StartedDateTime time.Time
	Time            int64
	Request         HarRequest
	Response        HarResponse
	Timings         HarTimings
	ServerIpAddress string
	Connection      string
}

type HarRequest struct {
	Method         string				`json:"method"`
	Url            string				`json:"url"`
	HttpVersion    string				`json:"httpVersion"`
	Cookies        []HarCookie			`json:"cookies"`
	Headers        []HarNameValuePair	`json:"headers"`
	QueryString    []HarNameValuePair	`json:"queryString"`
	PostData       HarPostData			`json:"postData"`
	BodySize       int64				`json:"bodySize"`
	HeadersSize    uintptr					`json:"headersSize"`
	RequestBody    string				`json:"requestBody"`
}


func ParseRequest(req *http.Request) *HarRequest {
	harRequest := HarRequest {
		Method 		: req.Method,
		Url    		: req.URL.String(),
		HttpVersion : req.Proto,
		QueryString : parseStringArrMap((req.URL.Query())),
		Cookies 	: parseCookies(req.Cookies()),
		Headers		: parseStringArrMap(req.Header),
		HeadersSize : -1,
	}

	return &harRequest
}

func parseStringArrMap(stringArrMap map[string][]string) []HarNameValuePair {
	index := 0
	harQueryString := make([]HarNameValuePair, len(stringArrMap))
	for k, v := range stringArrMap {
		harNameValuePair := HarNameValuePair {
			Name  : k,
			Value : strings.Join(v, ","),
		}
		harQueryString[index] = harNameValuePair
		index++
	}
	return harQueryString
}

func parseCookies(cookies []*http.Cookie) []HarCookie {
	harCookies := make([]HarCookie, len(cookies))
	for i, cookie := range cookies {
		harCookie := HarCookie {
			Name 	 : cookie.Name,
			Domain 	 : cookie.Domain,
			Expires  : cookie.Expires,
			HttpOnly : cookie.HttpOnly,
			Path 	 : cookie.Path,
			Secure 	 : cookie.Secure,
			Value 	 : cookie.Value,
	}
		harCookies[i] = harCookie
	}
	return harCookies
}


func (hr *HarRequest) String() string {
	str, _ := json.MarshalIndent(hr, "", "\t")
	return string(str)
}

type HarResponse struct {
	Status             int
	StatusText         string
	HttpVersion        string
	Cookies            []HarCookie
	Headers            []HarNameValuePair
	Content            HarContent
	RedirectUrl        string
	BodySize           int64
	HeadersSize        int64
}

type HarCookie struct {
	Name string
	Value string
	Path string
	Domain string
	Expires time.Time
	HttpOnly bool
	Secure bool
}

type HarNameValuePair struct {
	Name string
	Value string
}

type HarPostData struct {
	MimeType string
	Params   []HarPostDataParam
	Text     string
}

type HarPostDataParam struct {
	Name        string
	Value       string
	FileName    string
	ContentType string
}

type HarContent struct {
	Size        int64
	Compression int64
	MimeType    string
	Text        string
	Encoding    string
}

type HarPageTimings struct {
	OnContentLoad int64
	OnLoad int64
}

type HarTimings struct {
	Blocked int64
	Dns int64
	Connect int64
	Send int64
	Wait int64
	Receive int64
	Ssl int64
}


