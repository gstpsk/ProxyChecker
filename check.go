package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

func checkProxy(proxyString string, urlString string) (bool, int) {
	proxyUrl := parseProxyURL(proxyString)

	client := createNewHTTPClient(proxyUrl)

	// Create the HTTP GET request
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		log.Printf("Failed to create HTTP GET request: %s", err)
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	timestampStart := time.Now().UnixNano() / int64(time.Millisecond)
	// Send the request
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		log.Printf("HTTP GET request failed: %s", err)
		return false, 0
	}
	defer resp.Body.Close()
	timestampFinish := time.Now().UnixNano() / int64(time.Millisecond)
	timeDiff := timestampFinish - timestampStart
	fmt.Printf("%dms", timeDiff)

	if resp.StatusCode != 418 {
		return false, int(timeDiff)
	}

	return true, int(timeDiff)
}

// Returns a URL object when given a string
func parseProxyURL(proxyString string) *url.URL {
	proxyUrl, err := url.Parse("http://" + proxyString)
	if err != nil {
		log.Printf("Invalid proxy URL: %s", err)
	}
	return proxyUrl
}

// Returns a custom HTTP client
func createNewHTTPClient(proxyUrl *url.URL) *http.Client {
	// Apply transport settings
	tr := &http.Transport{
		DisableKeepAlives:  true,
		Proxy:              http.ProxyURL(proxyUrl),
		ProxyConnectHeader: http.Header{},
	}

	client := &http.Client{Transport: tr} // , Timeout: timeout
	return client
}
