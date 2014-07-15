package goharproxy

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"net/url"
	"crypto/tls"
	"io"
	"log"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"bytes"
	"io/ioutil"
	"strings"
)

var acceptAllCerts = &tls.Config{InsecureSkipVerify: true}
var srv = httptest.NewServer(nil)

func init() {
	http.DefaultServeMux.Handle("/bobo", ConstantHanlder("bobo"))
	http.DefaultServeMux.Handle("/query", QueryHandler{})
	http.DefaultServeMux.Handle("/", ConstantHanlder("google"))
}

type QueryHandler struct{}

func (QueryHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		panic(err)
	}
	io.WriteString(w, req.Form.Get("result"))
}

type ConstantHanlder string

func (h ConstantHanlder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, string(h))
}
// HarProxy Tests

func TestHttpHarProxyGetIpEntries(t *testing.T) {
	client, harProxy, s := oneShotProxy()
	defer s.Close()

	testUrl , _ := url.Parse(srv.URL + "/bobo")
	_, err := client.Get(testUrl.String())
	if err != nil {
		t.Fatal(err)
	}
	harLog := testLog(t, harProxy.NewHarReader())
	host, _, _ := net.SplitHostPort(testUrl.Host)
	testIpAddr(t, host, harLog)
}

func TestHttpHarProxyGetHostEntries(t *testing.T) {
	client, harProxy, s := oneShotProxy()
	defer s.Close()

	_, err := client.Get("http://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	ipaddr, ipErr := net.LookupIP("www.google.com")
	if ipErr != nil {
		t.Fatal(ipErr)
	}
	if len(ipaddr) == 0 {
		t.Fatal("Didn't get ip for www.google.com")
	}
	harLog := testLog(t, harProxy.NewHarReader())
	testIpAddr(t, ipaddr[0].String(), harLog)
}

func testIpAddr(t *testing.T, expected string, harLog *HarLog) {
	actual := harLog.Entries[0].ServerIpAddress
	if actual != expected {
		t.Fatal("Expected to get host: ", expected, " but got: ", actual)
	}
}

// HarProxyServer tests

func TestHarProxyServerGetProxyAndDelete(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerPort, _ := getProxiedClient(t, harProxyServer, testClient)
	proxyServerDeleteUrl := fmt.Sprintf("%v/proxy/%v", harProxyServer.URL, proxyServerPort.Port)
	req , reqErr := http.NewRequest("DELETE", proxyServerDeleteUrl, nil)
	if reqErr != nil {
		t.Fatal(reqErr)
	}
	resp, respErr := testClient.Do(req)
	testResp(t, resp, respErr)

	var proxyServerMessage *ProxyServerMessage = new(ProxyServerMessage)
	json.NewDecoder(resp.Body).Decode(proxyServerMessage)

	if proxyServerMessage.Message != fmt.Sprintf("Deleted proxy for port [%v] succesfully", proxyServerPort.Port) {
		t.Fatal("Did not get delete message")
	}
}

func TestHarProxyServerSendInvalidMessage(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerUrl := fmt.Sprintf("%v/bla", harProxyServer.URL)
	resp , err := testClient.Get(proxyServerUrl)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatal("Did not get 404 status code")
	}

	var proxyErrorMessage *ProxyServerErr = new(ProxyServerErr)
	json.NewDecoder(resp.Body).Decode(proxyErrorMessage)

	if proxyErrorMessage.Error != fmt.Sprintf("No such path: [%v]", "/bla") {
		t.Fatal("Did not get expected error message")
	}
}

func TestHarProxyServerGetInvalidProxy(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerHarUrl := fmt.Sprintf("%v/proxy/%v/har", harProxyServer.URL, 9999)
	req , err := http.NewRequest("PUT", proxyServerHarUrl, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, respErr := testClient.Do(req)
	if respErr != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatal("Did not get 404 status code")
	}

	var proxyErrorMessage *ProxyServerErr = new(ProxyServerErr)
	json.NewDecoder(resp.Body).Decode(proxyErrorMessage)

	if proxyErrorMessage.Error != fmt.Sprintf("No proxy for port [%v]", 9999) {
		t.Fatal("Did not get expected error message")
	}
}

func TestHarProxyServerSendInvalidProxyMessage(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerPort, _ := getProxiedClient(t, harProxyServer, testClient)
	proxyServerHarUrl := fmt.Sprintf("%v/proxy/%v/bla", harProxyServer.URL, proxyServerPort.Port)
	req , reqErr := http.NewRequest("PUT", proxyServerHarUrl, nil)
	if reqErr != nil {
		t.Fatal(reqErr)
	}
	resp, respErr := testClient.Do(req)
	if respErr != nil {
		t.Fatal(respErr)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatal("Did not get 404 status code")
	}

	var proxyErrorMessage *ProxyServerErr = new(ProxyServerErr)
	json.NewDecoder(resp.Body).Decode(proxyErrorMessage)

	if proxyErrorMessage.Error != "No such path [/bla] with method PUT" {
		t.Fatal("Did not get expected error message")
	}
}

func TestHarProxyServerGetProxyAndEntries(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerPort, proxiedClient := getProxiedClient(t, harProxyServer, testClient)
	_, err := proxiedClient.Get(srv.URL + "/bobo")
	if err != nil {
		t.Fatal(err)
	}

	proxyServerHarUrl := fmt.Sprintf("%v/proxy/%v/har", harProxyServer.URL, proxyServerPort.Port)
	req , reqErr := http.NewRequest("PUT", proxyServerHarUrl, nil)
	if reqErr != nil {
		t.Fatal(reqErr)
	}
	resp, respErr := testClient.Do(req)
	testResp(t, resp, respErr)
	testLog(t, resp.Body)
}

func TestHarProxyServerGetProxyAndEntriesWithResponseContent(t *testing.T) {
	captureContent = true
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerPort, proxiedClient := getProxiedClient(t, harProxyServer, testClient)
	resp, err := proxiedClient.Get(srv.URL + "/query?result=bla")
	if err != nil {
		t.Fatal(err)
	}
	txt, err  := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Error reading response body")
	}
	if string(txt) != "bla" {
		t.Fatal("Expect to get bla in response body")
	}

	proxyServerHarUrl := fmt.Sprintf("%v/proxy/%v/har", harProxyServer.URL, proxyServerPort.Port)
	req , reqErr := http.NewRequest("PUT", proxyServerHarUrl, nil)
	if reqErr != nil {
		t.Fatal(reqErr)
	}
	resp, err = testClient.Do(req)
	testResp(t, resp, err)

	harLog := testLog(t, resp.Body)
	if harLog.Entries[0].Response.Content.Text != "bla" {
		t.Fatal("Expect to get bla in har response")
	}

}

func TestHarProxyServerGetProxyAndEntriesWithRequestPostData(t *testing.T) {
	captureContent = true
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerPort, proxiedClient := getProxiedClient(t, harProxyServer, testClient)
	resp, err := proxiedClient.Post(srv.URL + "/bobo", "form-data", strings.NewReader("bla"))
	if err != nil {
		t.Fatal(err)
	}

	proxyServerHarUrl := fmt.Sprintf("%v/proxy/%v/har", harProxyServer.URL, proxyServerPort.Port)
	req , reqErr := http.NewRequest("PUT", proxyServerHarUrl, nil)
	if reqErr != nil {
		t.Fatal(reqErr)
	}
	resp, err = testClient.Do(req)
	testResp(t, resp, err)

	harLog := testLog(t, resp.Body)
	if harLog.Entries[0].Request.PostData.Text!= "bla" {
		t.Fatal("Expect to get bla in har request")
	}

}

func TestHarProxyServerGetProxyChangeHost(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	proxyServerPort, proxiedClient := getProxiedClient(t, harProxyServer, testClient)
	proxyServerHostUrl := fmt.Sprintf("%v/proxy/%v/hosts", harProxyServer.URL, proxyServerPort.Port)

	srvUrl , _ := url.Parse(srv.URL)
	proxyHosts := []ProxyHosts{{Host : "www.google.com", NewHost : srvUrl.Host}}
	proxyHostsJson, _ := json.Marshal(&proxyHosts)
	buffer := bytes.NewBuffer(proxyHostsJson)
	_, err := testClient.Post(proxyServerHostUrl, "application/json", buffer)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := proxiedClient.Get("http://www.google.com")
	testResp(t, resp, err)

	str, _ := ioutil.ReadAll(resp.Body)
	if string(str) != "google" {
		t.Fatal("Failed redirecting request")
	}
}

func getProxiedClient(t *testing.T, harProxyServer *httptest.Server, testClient *http.Client) (proxyServerPort *ProxyServerPort, client *http.Client) {
	resp, err := testClient.Post(harProxyServer.URL + "/proxy", "", nil)
	testResp(t, resp, err)

	proxyServerPort = new(ProxyServerPort)
	if e := json.NewDecoder(resp.Body).Decode(proxyServerPort); e != nil {
		log.Fatal(e)
	}

	host, _, _ := net.SplitHostPort(harProxyServer.URL)
	proxyUrl, _ := url.Parse(host + ":" + strconv.Itoa(proxyServerPort.Port))
	client = newProxyHttpTestClient(proxyUrl)
	return
}

func testLog(t *testing.T, r io.Reader) *HarLog{
	var harLog *HarLog = new(HarLog)
	json.NewDecoder(r).Decode(harLog)
	log.Printf("Har entries len: %v", len(harLog.Entries))
	if len(harLog.Entries) == 0 {
		t.Fatal("Didn't get valid har entries")
	}
	return harLog
}

func testResp(t *testing.T, resp *http.Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatal(resp.Status)
	}
}

func oneShotProxy() (client *http.Client, harProxy *HarProxy, s *httptest.Server) {
	harProxy = NewHarProxy()
	client, s = newProxyHttpTestServer(harProxy)
	return
}

func newProxyTestServer() (client *http.Client, s *httptest.Server) {
	s = httptest.NewServer(http.HandlerFunc(proxyHandler))

	tr := &http.Transport{TLSClientConfig: acceptAllCerts}
	client = &http.Client{Transport: tr}
	return
}

func newProxyHttpTestServer(harProxy *HarProxy) (client *http.Client, s *httptest.Server) {
	s = httptest.NewServer(harProxy.Proxy)
	proxyUrl, _ := url.Parse(s.URL)
	client = newProxyHttpTestClient(proxyUrl)
	return
}

func newProxyHttpTestClient(proxyUrl *url.URL) (client *http.Client) {
	log.Println(proxyUrl)
	tr := &http.Transport{TLSClientConfig: acceptAllCerts, Proxy: http.ProxyURL(proxyUrl)}
	client = &http.Client{Transport: tr}
	return
}
