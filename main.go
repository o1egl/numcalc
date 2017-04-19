package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxURLRuneCount = 2083
	MinURLRuneCount = 11

	IP           string = `(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`
	URLSchema    string = `((https?):\/\/)`
	URLUsername  string = `(\S+(:\S*)?@)`
	URLPath      string = `((\/|\?|#)[^\s]*)`
	URLPort      string = `(:(\d{1,5}))`
	URLIP        string = `([1-9]\d?|1\d\d|2[01]\d|22[0-3])(\.(1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.([0-9]\d?|1\d\d|2[0-4]\d|25[0-4]))`
	URLSubdomain string = `((www\.)|([a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*))`
	URL          string = `^` + URLSchema + URLUsername + `?` + `((` + URLIP + `|(\[` + IP + `\])|(([a-zA-Z0-9]([a-zA-Z0-9-]+)?[a-zA-Z0-9]([-\.][a-zA-Z0-9]+)*)|(` + URLSubdomain + `?))?(([a-zA-Z\x{00a1}-\x{ffff}0-9]+-?-?)*[a-zA-Z\x{00a1}-\x{ffff}0-9]+)(?:\.([a-zA-Z\x{00a1}-\x{ffff}]{1,}))?))` + URLPort + `?` + URLPath + `?$`
)

var (
	rxURL       = regexp.MustCompile(URL)
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
		executionTimeout = 500 * time.Millisecond
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

		go NumbersProcessor(ctx, numbersCh, responseCh)

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
	}
}

// NumbersProcessor computes NumbersResp
func NumbersProcessor(ctx context.Context, inChan chan []int, outChan chan []byte) {
	numbersSet := make(map[int]struct{})
	calculatedNumbers := NumbersResp{Numbers: []int{}}

	for {
		select {
		case resp := <-inChan:
			for _, n := range resp {
				if _, ok := numbersSet[n]; !ok {
					numbersSet[n] = struct{}{}
					calculatedNumbers.Numbers = append(calculatedNumbers.Numbers, n)
				}
			}
			sort.Ints(calculatedNumbers.Numbers)
			b, err := json.Marshal(&calculatedNumbers)
			if err != nil {
				outChan <- nil
				log.Println(err)
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
		outChan <- nil
		log.Printf("Error creating request from %s", u)
		return
	}
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		outChan <- nil
		log.Printf("Error %s: %s", u, err.Error())
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
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
	select {
	case <-ctx.Done():
		log.Printf("Error %s: %s", u, ctx.Err())
	case outChan <- numbers.Numbers:
		return
	}
}

// IsValidURL check if the string is an URL.
func IsValidURL(str string) bool {
	if str == "" || utf8.RuneCountInString(str) >= MaxURLRuneCount || len(str) <= MinURLRuneCount || strings.HasPrefix(str, ".") {
		return false
	}
	u, err := url.Parse(str)
	if err != nil {
		return false
	}
	if strings.HasPrefix(u.Host, ".") {
		return false
	}
	if u.Host == "" && (u.Path != "" && !strings.Contains(u.Path, ".")) {
		return false
	}
	return rxURL.MatchString(str)

}
