package main

import (
	"log"
	"sync"
)

var workerGroup = &sync.WaitGroup{}

func worker() {
	defer workerGroup.Done()
	for proxy := range inProxyChan {
		result, delay := checkProxy(proxy, UrlString)
		resultObj := ProxyResult{
			address: proxy,
			working: result,
			delay:   delay,
		}
		resultsChan <- resultObj
	}
}

func startWorkersAndWait() {
	for i := 0; i < WorkerCount; i++ {
		workerGroup.Add(1)
		go worker()
	}

	workerGroup.Wait()
	log.Println("All workers finished")
}
