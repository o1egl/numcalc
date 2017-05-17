package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"time"
)

var (
	emptyResult []byte
)

// NumbersResp represents NumbersHandler's response
type NumbersResp struct {
	Numbers []int `json:"numbers"`
}

func main() {
	listenAddr := flag.String("http.addr", ":8080", "http listen address")
	flag.Parse()

	var (
		err              error
		executionTimeout = 460 * time.Millisecond
	)
	emptyResult, err = json.Marshal(&NumbersResp{Numbers: []int{}})
	if err != nil {
		log.Fatalln(err)
	}

	http.HandleFunc("/numbers", NumbersHandler(executionTimeout))
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}

// NumbersHandler calculates NumbersResp from received urls
// executionTimeout: declares handler's timeout
// Example request `curl http://localhost:8080/numbers?u=http://localhost:8090/primes&u=http://localhost:8090/rand`
func NumbersHandler(executionTimeout time.Duration) func(resp http.ResponseWriter, req *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancelFunc := context.WithTimeout(context.Background(), executionTimeout)
		urls := req.URL.Query()["u"]
		responseCh := make(chan []byte, len(urls))
		numbersCh := make(chan []int, len(urls))
		responseBody := emptyResult

		forProcess := 0
		processed := 0
		for i := 0; i < len(urls); i++ {
			if IsValidURL(urls[i]) {
				forProcess++
				go NumbersExtractor(ctx, urls[i], numbersCh)
			}
		}

		if forProcess == 0 {
			cancelFunc()
			resp.Write(responseBody)
			return
		}

		go NumbersProcessor(ctx, len(urls), numbersCh, responseCh)

	Loop:
		for {
			select {
			case <-ctx.Done():
				resp.Write(responseBody)
				break Loop
			case responseBody = <-responseCh:
				processed++
				if forProcess == processed {
					resp.Write(responseBody)
					break Loop
				}
			}
		}
		cancelFunc()
	}
}

// NumbersProcessor computes NumbersResp
func NumbersProcessor(ctx context.Context, possibleRespCount int, inChan chan []int, outChan chan []byte) {
	responses := make([][]int, possibleRespCount)

	for {
		select {
		case resp := <-inChan:
			responses = append(responses, resp)

			b, err := json.Marshal(&NumbersResp{Numbers: MergeAndDedup(responses...)})
			if err != nil {
				log.Println(err)
				outChan <- nil
			} else {
				outChan <- b
			}

		case <-ctx.Done():
			return
		}
	}
}

// NumbersExtractor process incoming url and send result to outChan
func NumbersExtractor(ctx context.Context, u string, outChan chan []int) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Printf("Error creating request from %s. %s", u, err.Error())
		outChan <- nil
		return
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		outChan <- nil
		log.Printf("Error %s: %s", u, err.Error())
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		outChan <- nil
		log.Printf("Error %s: %s", u, err.Error())
		return
	}

	var numbers NumbersResp
	err = json.Unmarshal(body, &numbers)
	if err != nil {
		outChan <- nil
		log.Printf("Error %s: %s", u, err.Error())
		return
	}
	sort.Ints(numbers.Numbers)
	select {
	case <-ctx.Done():
		log.Printf("Error %s: %s", u, ctx.Err())
	case outChan <- numbers.Numbers:
		return
	}
}
