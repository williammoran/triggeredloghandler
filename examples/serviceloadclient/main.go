package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func main() {
	useNegative := flag.Bool("n", false, "Submit negative numbers")
	useInvalid := flag.Bool("i", false, "Submit invalid numbers")
	threads := flag.Int("t", 1, "Number of parallel threads to run")
	requests := flag.Int("r", 1000, "Requests per thread")
	url := flag.String("u", "http://localhost:8000/sqrt/", "URL to GET")
	flag.Parse()
	wg := sync.WaitGroup{}
	start := time.Now()
	for t := 0; t < *threads; t++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			makeRequests(*url, *requests, *useNegative, *useInvalid)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("use negative = %t use invalid = %t to URL %s\n", *useNegative, *useInvalid, *url)
	fmt.Printf("%d threads %d requests each took %s\n", *threads, *requests, elapsed)
}

func makeRequests(url string, c int, useNeg, useInv bool) {
	for i := 0; i < c; i++ {
		valType := rand.Intn(3)
		val := fmt.Sprintf("%f", rand.Float64())
		switch valType {
		case 0:
			// Already using valid number
		case 1:
			if useNeg {
				val = "-1"
			}
		case 2:
			if useInv {
				val = "NotANumber"
			}
		}
		resp, err := http.Get(url + val)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if resp.StatusCode != 200 {
			log.Println(resp.Status)
			continue
		}
		_, err = strconv.ParseFloat(string(body), 64)
		if err != nil {
			log.Println(err.Error())
			continue
		}
	}
}
