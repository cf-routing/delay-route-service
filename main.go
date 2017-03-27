package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	DEFAULT_PORT              = "8080"
	CF_FORWARDED_URL_HEADER   = "X-Cf-Forwarded-Url"
	CF_PROXY_SIGNATURE_HEADER = "X-Cf-Proxy-Signature"
)

func main() {
	var (
		port              string
		skipSslValidation bool
		err               error
	)

	if port = os.Getenv("PORT"); len(port) == 0 {
		port = DEFAULT_PORT
	}
	if skipSslValidation, err = strconv.ParseBool(os.Getenv("SKIP_SSL_VALIDATION")); err != nil {
		skipSslValidation = true
	}
	log.SetOutput(os.Stdout)

	roundTripper := NewDelayRoundTripper(skipSslValidation)
	proxy := NewProxy(roundTripper, skipSslValidation)

	log.Fatal(http.ListenAndServe(":"+port, proxy))
}

func NewProxy(transport http.RoundTripper, skipSslValidation bool) http.Handler {
	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			forwardedURL := req.Header.Get(CF_FORWARDED_URL_HEADER)

			logRequest(req.Header, skipSslValidation)

			// Note that url.Parse is decoding any url-encoded characters.
			url, err := url.Parse(forwardedURL)
			if err != nil {
				log.Fatalln(err.Error())
			}

			req.URL = url
			req.Host = url.Host
			delayRequest(req)
		},
		Transport:     transport,
		FlushInterval: 50 * time.Millisecond,
	}

	return reverseProxy
}

func logRequest(headers http.Header, skipSslValidation bool) {
	log.Printf("Skip ssl validation set to %t", skipSslValidation)
	log.Println("Received request: ")
	log.Println("")
	log.Printf("Headers: %#v\n", headers)
	log.Println("")
}

type DelayRoundTripper struct {
	transport http.RoundTripper
}

func NewDelayRoundTripper(skipSslValidation bool) http.RoundTripper {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSslValidation},
	}
	return &DelayRoundTripper{
		transport: tr,
	}
}

func (lrt *DelayRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	sleep()
	res, err := lrt.transport.RoundTrip(request)
	log.Printf("Request has completed %#v\n", res)
	// if request.URL.Path == "/routing/v1/tcp_routes" {
	// }

	return res, err
}

func delayRequest(req *http.Request) {
	if req.URL.Path == "/routing/v1/tcp_routes" {
		sleep()
	}
}

func sleep() {
	// sleepMilliString := os.Getenv("ROUTE_SERVICE_SLEEP_MILLI")
	// sleepMilli, _ := strconv.ParseInt(sleepMilliString, 0, 64)
	log.Printf("Sleeping for %d milliseconds\n", 30)
	time.Sleep(time.Duration(30) * time.Second)
}
