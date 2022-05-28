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

var urlString string = "http://www.httpbin.org/status/418"
var timeout time.Duration = 20 * time.Second
var inProxyChan = make(chan string, 1000)
var wg = &sync.WaitGroup{}
var workerCount int = 30
var proxyTotal int = 0
var proxyFile string = "proxies.txt"
var resultsChan = make(chan ProxyResult, 1000)
var workingProxies = []ProxyResult{}
var showLogs = false

type ProxyResult struct {
	address string
	working bool
	delay   int
}

func main() {
	proxyTotal = countProxies(proxyFile)

	if !showLogs {
		f, _ := os.Open("/dev/null")
		log.SetOutput(f)
	}

	go proxyInput(proxyFile)

	fmt.Printf("Checking %d proxies with %d threads...", proxyTotal, workerCount)

	wg.Add(1)
	go printProgress()

	startWorkersAndWait()

	close(resultsChan)

	wg.Wait() // Wait for printProgress to finish
}

// Literally opens the file and counts each line
func countProxies(filename string) int {
	f, err := os.Open(filename) // Get file
	if err != nil {
		log.Fatalf("Failed to open proxy file: %s", err)
	}
	defer f.Close()

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
	if err != nil {
		log.Fatalf("Couldn't read from proxy file: %s", err)
	}
	defer f.Close()

	// Create new reader
	reader := bufio.NewReader(f)

	// Read from file
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		inProxyChan <- string(line)
	}
}

// Prints progress from resultsChan
func printProgress() {
	defer wg.Done()
	var checked int

	f, err := os.Create(fmt.Sprintf("proxy-results-%d-%d-%d-%d-%d-%d.txt", time.Now().Year(), time.Now().Month(), time.Now().Day(), time.Now().Hour(), time.Now().Minute(), time.Now().Second()))
	if err != nil {
		log.Fatalf("Failed to write output to file: %s", err)
	}
	defer f.Close()

	f.WriteString("ADDRESS					DELAY (ms)\n")

	for result := range resultsChan {
		checked++
		if result.working {
			workingProxies = append(workingProxies, result)
			f.WriteString(fmt.Sprintf("%s		%d\n", result.address, result.delay))
		}
		fmt.Printf("\nTotal:%d Checked:%d Working:%d Success rate:%d%%", proxyTotal, checked, len(workingProxies), int(math.Round(float64(len(workingProxies))/float64(checked)*100)))
	}
}
