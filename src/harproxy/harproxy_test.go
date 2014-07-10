package harproxy

import (
	"testing"
	"net/http"
	"net/http/httptest"
	"net/url"
	"crypto/tls"
	"io"
	"log"
	"encoding/json"
	"har"
	"fmt"
	"net"
	"strconv"
)

var acceptAllCerts = &tls.Config{InsecureSkipVerify: true}
var srv = httptest.NewServer(nil)

func init() {
	http.DefaultServeMux.Handle("/bobo", ConstantHanlder("bobo"))
}

type ConstantHanlder string

func (h ConstantHanlder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("bobo")
	io.WriteString(w, string(h))
}
// HarProxy Tests

func TestHttpHarProxyGetEntries(t *testing.T) {
	client, harProxy, s := oneShotProxy()
	defer s.Close()

	_, err := client.Get(srv.URL + "/bobo")
	if err != nil {
		t.Fatal(err)
	}
	var harEntries *[]har.HarEntry = new([]har.HarEntry)
	json.NewDecoder(harProxy.NewHarReader()).Decode(harEntries)
	log.Printf("Har entries len: %v", len(*harEntries))
	if len(*harEntries) == 0 {
		t.Fatal("Didn't get valid har entries")
	}
}

// HarProxyServer tests

func TestHarProxyServerGetProxyAndEntries(t *testing.T) {
	testClient, harProxyServer := newProxyTestServer()
	defer harProxyServer.Close()

	resp, err := testClient.Post(harProxyServer.URL + "/proxy", "", nil)
	testResp(t, resp, err)

	var proxyServerPort *ProxyServerPort = new(ProxyServerPort)
	if e := json.NewDecoder(resp.Body).Decode(proxyServerPort); e != nil {
		log.Fatal(e)
	}

	host, _, _ := net.SplitHostPort(harProxyServer.URL)
	proxyUrl, _ := url.Parse(host + ":" + strconv.Itoa(proxyServerPort.Port))
	proxiedClient := newProxyHttpTestClient(proxyUrl)
	_, err = proxiedClient.Get(srv.URL + "/bobo")
	if err != nil {
		t.Fatal(err)
	}

	proxyServerHarUrl := fmt.Sprintf("%v/proxy/%v/har", harProxyServer.URL, proxyServerPort.Port)
	req , reqErr := http.NewRequest("PUT", proxyServerHarUrl, nil)
	if reqErr != nil {
		t.Fatal(err)
	}
	resp, err = testClient.Do(req)
	testResp(t, resp, err)

	var harEntries *[]har.HarEntry = new([]har.HarEntry)
	json.NewDecoder(resp.Body).Decode(harEntries)
	log.Printf("Har entries len: %v", len(*harEntries))
	if len(*harEntries) == 0 {
		t.Fatal("Didn't get valid har entries")
	}
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
