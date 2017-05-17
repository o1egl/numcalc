package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestNumbersProcessor(t *testing.T) {
	expected, _ := json.Marshal(&NumbersResp{
		Numbers: []int{1, 3, 5, 7, 8, 10, 12, 16},
	})
	numbers1 := []int{1, 3, 5, 8, 12}
	numbers2 := []int{1, 3, 7, 8, 10, 16}
	out := make(chan []byte)
	in := make(chan []int, 2)

	go NumbersProcessor(context.Background(), 2, in, out)
	in <- numbers1
	in <- numbers2
	close(in)

	obtained := <-out
	obtained = <-out
	if bytes.Compare(expected, obtained) != 0 {
		t.Errorf("Expected: %s \n Obtained: %s", string(expected), string(obtained))
	}
}

func TestNumbersExtractor(t *testing.T) {
	numbersCh := make(chan []int)

	expected := []int{2, 3, 5, 7, 11, 13}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]interface{}{"numbers": expected})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	go NumbersExtractor(context.Background(), server.URL, numbersCh)

	obtained := <-numbersCh

	for i := range expected {
		if expected[i] != obtained[i] {
			t.Errorf("Expected: %v \n Obtained: %v", expected, obtained)
		}
	}
}

func TestNumbersExtractorTimeout(t *testing.T) {
	numbersCh := make(chan []int)

	expected := []int{2, 3, 5, 7, 11, 13}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]interface{}{"numbers": expected})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
	go NumbersExtractor(ctx, server.URL, numbersCh)

	obtained := <-numbersCh

	if len(obtained) > 0 {
		t.Errorf("Expected: %v \n Obtained: %v", []int{}, obtained)
	}
}

func TestNumbersHandler(t *testing.T) {
	server := httptest.NewServer(testHandler(map[string]struct {
		Numbers []int
		Sleep   time.Duration
	}{
		"/one": {Numbers: []int{5, 3, 6}},
		"/two": {Numbers: []int{10, 3, 7}},
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", "/health-check?u="+server.URL+"/one&&u="+server.URL+"/two", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(NumbersHandler(500 * time.Millisecond))
	start := time.Now()
	handler.ServeHTTP(rr, req)

	executionTime := time.Since(start).Nanoseconds() / int64(time.Millisecond)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"numbers":[3,5,6,7,10]}`

	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	if executionTime > 499 {
		t.Errorf("execution time is %d ms byt should be less than 500 ms", executionTime)
	}
}

func TestNumbersHandlerTimeout(t *testing.T) {

	// give time for closing previous servers
	time.Sleep(2 * time.Second)

	numGoroutines := runtime.NumGoroutine()

	server := httptest.NewServer(testHandler(map[string]struct {
		Numbers []int
		Sleep   time.Duration
	}{
		"/one":   {Numbers: []int{5, 3, 6}},
		"/two":   {Numbers: []int{10, 3, 7}},
		"/three": {Numbers: []int{1, 8, 3}, Sleep: 1 * time.Second},
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", "/health-check?u="+server.URL+"/one&&u="+server.URL+"/two&u="+server.URL+"/three", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(NumbersHandler(500 * time.Millisecond))
	start := time.Now()
	handler.ServeHTTP(rr, req)

	executionTime := time.Since(start).Nanoseconds() / int64(time.Millisecond)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := `{"numbers":[3,5,6,7,10]}`

	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}

	if executionTime > 508 {
		t.Errorf("handler's execution time is %d ms but should be less than 500 ms", executionTime)
	}

	server.Close()

	// test goroutine leak
	numGoroutines2 := runtime.NumGoroutine()
	if numGoroutines != numGoroutines2 {
		t.Errorf("Leaked %d goroutine(s)", numGoroutines2-numGoroutines)
	}
}

func TestNumbersHandlerWrongUrls(t *testing.T) {
	expected := `{"numbers":[]}`

	req, err := http.NewRequest("GET", "/health-check?u=ftp://google.com&u=google.com", nil)
	if err != nil {
		t.Fatal(err)
	}

	emptyResult, _ = json.Marshal(&NumbersResp{Numbers: []int{}})

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(NumbersHandler(500 * time.Millisecond))
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestIsValidURL(t *testing.T) {
	testCases := []struct {
		url   string
		valid bool
	}{
		{
			url:   "http://google.com",
			valid: true,
		},
		{
			url:   "https://google.com",
			valid: true,
		},
		{
			url:   "http://google.com:80?param=value&param2=value2",
			valid: true,
		},
		{
			url:   "https//google.com",
			valid: false,
		},
		{
			url:   "https://.google.com",
			valid: false,
		},
		{
			url:   "http\\s://google.com/",
			valid: false,
		},
		{
			url:   "https://.com",
			valid: false,
		},
		{
			url:   "google.com",
			valid: false,
		},
	}

	for _, test := range testCases {
		if IsValidURL(test.url) != test.valid {
			t.Errorf("IsValidURL(%s) should be %b", test.url, test.valid)
		}
	}
}

func testHandler(routes map[string]struct {
	Numbers []int
	Sleep   time.Duration
}) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s := routes[r.URL.Path]
		time.Sleep(s.Sleep)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]interface{}{"numbers": s.Numbers})
	})
}
