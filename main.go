// The MIT License (MIT)

// Copyright (c) 2015 Christopher Lillthors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"sync"
	"time"
)

var (
	numPar    int
	word      string
	wordMatch *regexp.Regexp
)

type result struct {
	url     string
	numFind int
}

type fetcher struct {
	inChan  chan string
	outChan chan *result
	errChan chan error
	wg      *sync.WaitGroup
}

func init() {
	flag.IntVar(&numPar, "numpar", 4, "Number of concurrent requests")
	flag.StringVar(&word, "find", "", "Word to find")
	// Parse all the flag options.
	flag.Parse()
	// Only look for whole words
	wordMatch = regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	// Use maximum number of cores.
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	// Check number of urls
	numRequests := len(flag.Args())
	// If number of urls == 0 OR word is empty
	if numRequests == 0 || word == "" {
		fmt.Fprintln(os.Stderr, "Please add number of concurrent requests and a word to find")
		return
	}
	// Create a new fetcher.
	f := &fetcher{
		inChan:  make(chan string, numRequests),
		outChan: make(chan *result, numRequests),
		errChan: make(chan error, numRequests),
		wg:      &sync.WaitGroup{},
	}
	// Wait for all the requests
	defer f.wg.Wait()
	// How many requests in total?
	f.wg.Add(numRequests)

	// Send arguments to worker routines.
	for _, urlString := range flag.Args() {
		u, err := url.Parse(urlString)
		if err != nil {
			f.errChan <- err
			continue
		}
		// Check if user added protocol for url
		if u.Scheme == "" {
			f.errChan <- fmt.Errorf("Please specify protocol for %s", urlString)
			continue
		}
		// Send raw url to worker pool
		f.inChan <- u.String()
	}

	// Spin up worker routines. One new worker/concurrent processes.
	for i := 0; i < numPar; i++ {
		// Create worker pool
		go f.worker()
	}

	// This will hold the result.
	var sum int
	for i := 0; i < numRequests; i++ {
		select {
		case res := <-f.outChan:
			// Add number of occurances.
			sum += res.numFind
			// Print out the result so far.
			fmt.Println(res)
		case err := <-f.errChan:
			// Print out error message
			fmt.Fprintln(os.Stderr, err)
		}
		// One request is done.
		f.wg.Done()
	}
	// Print out the sum of all occurances.
	fmt.Printf("Sum: \t\t%d\n", sum)
}

// Worker function.
func (f *fetcher) worker() {
	for {
		select {
		case url := <-f.inChan:
			// Create new http client.
			client := &http.Client{
				Timeout: time.Duration(10 * time.Second),
			}
			// do HTTP GET on url.
			res, err := client.Get(url)
			if err != nil {
				// Send back the error
				f.errChan <- err
				return
			}
			// Close the request.
			defer res.Body.Close()

			// Count the number of occurances, use stream to save memory.
			var count int
			for wordMatch.MatchReader(bufio.NewReader(res.Body)) {
				count++
			}
			// Send back the result.
			f.outChan <- &result{
				url:     url,
				numFind: count,
			}
		}
	}
}

func (r *result) String() string {
	return fmt.Sprintf("%s\t\t%d", r.url, r.numFind)
}
