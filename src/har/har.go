package har

import (
	"time"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"log"
	"io/ioutil"
)

type Har struct {
	HarLog HarLog	`json:"harLog"`
}

type HarLog struct {
	Pages   []HarPage		`json:"pages"`
	Entries []HarEntry		`json:"entries"`
}

type HarPage struct {
	Id              string			`json:"id"`
	StartedDateTime time.Time		`json:"startedDateTime"`
	Title           string			`json:"title"`
	PageTimings     HarPageTimings	`json:"pageTimings"`
}

type HarEntry struct {
	PageRef         string			`json:"pageRef"`
	StartedDateTime time.Time		`json:"startedDateTime"`
	Time            int64			`json:"time"`
	Request         HarRequest		`json:"request"`
	Response        HarResponse		`json:"response"`
	Timings         HarTimings		`json:"timings"`
	ServerIpAddress string			`json:"serverIpAddress"`
	Connection      string			`json:"connection"`
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
	HeadersSize    int64				`json:"headersSize"`
}

var captureContent bool = true

func ParseRequest(req *http.Request) *HarRequest {
	if req == nil {
		return nil
	}
	copyReq := *req
	harRequest := HarRequest {
		Method 		: copyReq.Method,
		Url    		: copyReq.URL.String(),
		HttpVersion : copyReq.Proto,
		Cookies 	: parseCookies(copyReq.Cookies()),
		Headers		: parseStringArrMap(copyReq.Header),
		QueryString : parseStringArrMap((copyReq.URL.Query())),
		BodySize	: copyReq.ContentLength,
		HeadersSize : -1,
	}

	if captureContent && (harRequest.Method == "POST" || harRequest.Method == "PUT") {
		harRequest.PostData = parsePostData(&copyReq)
	}

	return &harRequest
}

func parsePostData(req *http.Request) HarPostData {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("Error parsing request to %v: %v\n", req.URL, e)
		}
	}()
	e := req.ParseForm()
	if e != nil {
		panic(e)
	}
	harPostData := new(HarPostData)
	contentType := req.Header["Content-Type"]
	if contentType == nil {
		panic("Missing content type in request")
	}
	harPostData.MimeType = contentType[0]

	if len(req.PostForm) > 0 {
		index := 0
		params := make([]HarPostDataParam, len(req.PostForm))
		for k, v := range req.PostForm {
			param := HarPostDataParam {
				Name  : k,
				Value : strings.Join(v, ","),
			}
			params[index] = param
			index++
		}
		harPostData.Params = params
	} else {
		body, e := ioutil.ReadAll(req.Body)
		if e != nil {
			panic(e)
		}
		harPostData.Text = string(body)
	}
	return *harPostData
}


func parseStringArrMap(stringArrMap map[string][]string) []HarNameValuePair {
	index := 0
	harQueryString := make([]HarNameValuePair, len(stringArrMap))
	for k, v := range stringArrMap {
		escapedKey, _ 	 := url.QueryUnescape(k)
		escapedValues, _ := url.QueryUnescape(strings.Join(v, ","))
		harNameValuePair := HarNameValuePair {
			Name  : escapedKey,
			Value : escapedValues,
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
	Status             int					`json:"status"`
	StatusText         string				`json:"statusText"`
	HttpVersion        string				`json:"httpVersion`
	Cookies            []HarCookie			`json:"cookies"`
	Headers            []HarNameValuePair	`json:"headers"`
	Content            HarContent			`json:"content"`
	RedirectUrl        string				`json:"redirectUrl"`
	BodySize           int64				`json:"bodySize"`
	HeadersSize        int64				`json:"headersSize"`
}

func ParseResponse(resp *http.Response) *HarResponse {
	if resp == nil {
		return nil
	}
	harResponse := HarResponse {
		Status			: resp.StatusCode,
		StatusText		: resp.Status,
		HttpVersion		: resp.Proto,
		Cookies			: parseCookies(resp.Cookies()),
		Headers			: parseStringArrMap(resp.Header),
		RedirectUrl		: "",
		BodySize		: resp.ContentLength,
		HeadersSize		: -1,
	}

	if captureContent {

	}

	return &harResponse
}

func (hr *HarResponse) String() string {
	str, _ := json.MarshalIndent(hr, "", "\t")
	return string(str)
}

type HarCookie struct {
	Name     string			`json:"name"`
	Value    string			`json:"value"`
	Path     string			`json:"path"`
	Domain   string			`json:"domain"`
	Expires  time.Time		`json:"expires"`
	HttpOnly bool			`json:"httpOnly"`
	Secure   bool			`json:"secure"`
}

type HarNameValuePair struct {
	Name  string		`json:"name"`
	Value string		`json:"value"`
}

type HarPostData struct {
	MimeType string					`json:"mimeType"`
	Params   []HarPostDataParam		`json:"params"`
	Text     string					`json:"text"`
}

type HarPostDataParam struct {
	Name        string		`json:"name"`
	Value       string		`json:"value"`
	FileName    string		`json:"fileName"`
	ContentType string		`json:"contentType`
}

type HarContent struct {
	Size        int64		`json:"size"`
	Compression int64		`json:"compression"`
	MimeType    string		`json:"mimeType"`
	Text        string		`json:"text"`
	Encoding    string		`json:"encoding"`
}

type HarPageTimings struct {
	OnContentLoad int64		`json:"onContentLoad"`
	OnLoad        int64		`json:"onLoad"`
}

type HarTimings struct {
	Blocked int64
	Dns     int64
	Connect int64
	Send    int64
	Wait    int64
	Receive int64
	Ssl     int64
}


