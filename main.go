package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sync"
	"time"
)

var UrlString string = "http://www.httpbin.org/status/418"
var timeout time.Duration = 20 * time.Second
var results sync.Map
var ch = make(chan int, 1000) // big boi cuz idk
var inProxyChan = make(chan string, 1000)
var wg = &sync.WaitGroup{}
var WorkerCount int = 30
var proxyTotal int = 0
var proxyFile string = "proxies.txt"
var resultsChan = make(chan ProxyResult, 1000)
var workingProxies = []string{} // Create slice to store working proxies

type ProxyResult struct {
	address string
	working bool
	delay   int
}

func main() {
	proxyTotal = countProxies(proxyFile)

	go proxyInput(proxyFile)

	fmt.Printf("Checking %d proxies with %d threads...", proxyTotal, WorkerCount)

	wg.Add(1)
	go printProgress()

	startWorkersAndWait()

	close(resultsChan)
	close(ch)
	wg.Wait() // Wait for printProgress to finish

	outputToFile()
}

func countProxies(filename string) int {
	f, err := os.Open(filename) // Get file
	defer f.Close()
	if err != nil {
		log.Fatalf("Failed to open proxy file: %s", err)
	}

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

// Prints progress from resultsChan
func printProgress() {
	defer wg.Done()
	var counter int
	for result := range resultsChan {
		counter++
		var state string = "NOT WORKING"
		if result.working {
			state = "WORKING"
			workingProxies = append(workingProxies, result.address)
		}
		fmt.Printf("\n%s	%s		%dms\nProgress: %d/%d", result.address, state, result.delay, counter, proxyTotal)
		fmt.Printf("\nProxy success rate: %d/%d which is %d%%", len(workingProxies), counter, int(math.Round(float64(len(workingProxies))/float64(counter)*100)))
	}
}

func outputToFile() {
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
