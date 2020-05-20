package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/corpix/uarand"
)

var httpClient *http.Client = nil

func goRequest(r request) response {

	if httpClient == nil {

		proxyURL := http.ProxyFromEnvironment

		if len(r.proxy) > 0 {
			pu, err := url.Parse(r.proxy)
			if err == nil {
				proxyURL = http.ProxyURL(pu)
			}
		}
		var transport = &http.Transport{
			Proxy:             proxyURL,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: time.Second,
				DualStack: true,
			}).DialContext,
		}
		httpClient = &http.Client{
			Transport: transport,
		}
	}
	httpClient.Timeout = r.timeout

	if !r.followLocation {
		httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	var req *http.Request
	var err error
	if r.body != "" {
		req, err = http.NewRequest(r.method, r.URL(), bytes.NewBuffer([]byte(r.body)))
	} else {
		req, err = http.NewRequest(r.method, r.URL(), nil)
	}

	if err != nil {
		return response{request: r, err: err}
	}
	req.Close = true

	if !r.HasHeader("Host") {
		// add the host header to the request manually so it shows up in the output
		r.headers = append(r.headers, fmt.Sprintf("Host: %s", r.Hostname()))
	}

	if !r.HasHeader("User-Agent") {
		r.headers = append(r.headers, fmt.Sprintf("User-Agent: %s", userAgent))
	}

	for _, h := range r.headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) != 2 {
			continue
		}

		req.Header.Set(parts[0], parts[1])
	}

	req.Header.Set("User-Agent", uarand.GetRandom())

	resp, err := httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return response{request: r, err: err}
	}
	body, _ := ioutil.ReadAll(resp.Body)

	// extract the response headers
	hs := make([]string, 0)
	for k, vs := range resp.Header {
		for _, v := range vs {
			hs = append(hs, fmt.Sprintf("%s: %s", k, v))
		}
	}

	return response{
		request:    r,
		status:     resp.Status,
		statusCode: resp.StatusCode,
		headers:    hs,
		body:       body,
	}
}
