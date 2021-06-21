package limiter

import (
	"net/http/httptest"
	"net/http"
	"log"
	"io/ioutil"
	"io"
	"sync"
	"testing"

	"github.com/gorilla/mux"
)

var ( 
	mtx = &sync.Mutex{}
	successSlice = make([]int, 0)
	failedSlice = make([]int, 0)
	healthCheckHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"alive": true}`)
	}
)

func apicall(wg *sync.WaitGroup, url string) {
	defer wg.Done()

	resp, err := http.Get(url)
	
	if err != nil {
		panic(err)
	}		

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	bdy := string(body)
	log.Println(bdy)
	
	if bdy == "{\"alive\": true}" {
		mtx.Lock()
		successSlice = append(successSlice, 1)
		mtx.Unlock()
	} else {
		mtx.Lock()
		failedSlice = append(failedSlice, 1)
		mtx.Unlock()
	}
}

func Test_serverHandleFunc(t *testing.T) {
	r := mux.NewRouter()
	r.HandleFunc("/", healthCheckHandler)
	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")

	if err != nil {
		t.Errorf("Server: Expected nil, received %s", err.Error())
	}
	
	if res.StatusCode != http.StatusOK {
		t.Errorf("Server response: Expected %d, received %d", http.StatusOK, res.StatusCode)
	}
}

func Test_multipleApiCalls(t *testing.T) {
	var wg sync.WaitGroup

	r := mux.NewRouter()
	r.HandleFunc("/", healthCheckHandler)
	r.Use(LimiterMiddleware)
	ts := httptest.NewServer(r)
	defer ts.Close()

	for i := 1; i <= 15; i++ {
		wg.Add(1)
		go apicall(&wg, ts.URL + "/")
	}

	wg.Wait()

	if len(successSlice) != 10 {
		t.Errorf("Success Slice: Expected %d, received %d", 10, len(successSlice))
	}

	if len(failedSlice) != 5 {
		t.Errorf("Failed Slice: Expected %d, received %d", 10, len(failedSlice))
	}
}
