package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var urlString string = "http://www.httpbin.org/status/418"
var timeout time.Duration = 20 * time.Second
var results sync.Map
var ch = make(chan int, 1000) // big boi cuz idk
var inProxyChan = make(chan string, 1000)
var workerWait = &sync.WaitGroup{}
var wg = &sync.WaitGroup{}
var currency int = 30
var proxyTotal int = 0
var proxyFile string = "proxies.txt"
var resultsChan = make(chan CheckResult, 1000)
var workingProxies = []string{} // Create slice to store working proxies

type CheckResult struct {
	proxy  string
	result bool
}

func main() {
	// Load proxy list from file
	proxyTotal = countProxies(proxyFile)

	// Start proxy input thread
	go proxyInput(proxyFile)

	fmt.Printf("Checking %d proxies with %d threads...", proxyTotal, currency)

	// start workers
	for i := 0; i < currency; i++ {
		workerWait.Add(1)
		go worker()
	}

	wg.Add(1)
	go printProgress()

	// Wait for WaitGroup counters to reach zero
	workerWait.Wait()
	log.Print("all workers finished")
	close(resultsChan)
	close(ch)
	wg.Wait()
	log.Print("other boys aswell")

	// Open file
	f, err := os.Create("working.txt")
	if err != nil {
		log.Fatalf("Failed to write output to file: %s", err)
	}

	// Close file
	defer f.Close()

	// Write working proxies to file
	for _, proxy := range workingProxies {
		// Write string and newline
		f.WriteString(proxy + "\n")
	}
}

func worker() {
	defer workerWait.Done()
	for proxy := range inProxyChan {
		result := checkProxy(proxy, urlString)
		resultObj := CheckResult{
			proxy:  proxy,
			result: result,
		}
		resultsChan <- resultObj
	}
}

func countProxies(filename string) int {
	// Open proxy file
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		log.Fatalf("Failed to open proxy file: %s", err)
	}

	// Create reader
	reader := bufio.NewReader(f)
	var amount int = 0

	for {
		_, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return amount
			}
		} else {
			amount++
		}
	}
}

// proxyInput
//
// grabs proxy strings from a file and returns them via a channel
func proxyInput(filename string) {
	defer close(inProxyChan)

	// Open proxy file
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		log.Fatalf("Couldn't read from proxy file: %s", err)
	}

	// Create new reader
	reader := bufio.NewReader(f)

	// Read from file
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				log.Println("reached end")
				break
			}
		}
		inProxyChan <- string(line)
	}
}

func printProgress() {
	defer wg.Done()
	var counter int
	for result := range resultsChan {
		counter++
		var state string = "NOT WORKING"
		if result.result {
			state = "WORKING"
			workingProxies = append(workingProxies, result.proxy)
		}
		fmt.Printf("\n%s %s\nProgress: %d/%d", result.proxy, state, counter, proxyTotal)
		fmt.Printf("\nProxy success rate: %d/%d which is %d%%", len(workingProxies), counter, int(math.Round(float64(len(workingProxies))/float64(counter)*100)))
	}
}

func checkProxy(proxyString string, urlString string) bool {
	// Parse proxy string into URL
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
		return false
	}
	defer resp.Body.Close()
	timestampFinish := time.Now().UnixNano() / int64(time.Millisecond)
	timeDiff := timestampFinish - timestampStart
	fmt.Printf("%dms", timeDiff)

	if resp.StatusCode != 418 {
		return false
	}

	return true
}

func createNewHTTPClient(proxyUrl *url.URL) *http.Client {
	// Apply transport settings
	tr := &http.Transport{
		DisableKeepAlives:  true,
		Proxy:              http.ProxyURL(proxyUrl),
		ProxyConnectHeader: http.Header{},
	}

	// Add the transport object to HTTP client
	client := &http.Client{Transport: tr} // , Timeout: timeout
	return client
}

func parseProxyURL(proxyString string) *url.URL {
	// Parse proxy string into URL
	proxyUrl, err := url.Parse("http://" + proxyString)
	if err != nil {
		log.Printf("Invalid proxy URL: %s", err)
	}
	return proxyUrl
}

func basicAuth(username, password string) string {
	authString := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(authString))
}
