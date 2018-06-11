package loadbalancer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
)

type Worker struct {
	Host            string
	Address         string
	Port            int
	HealthcheckPath string
}

type Workers map[string][]Worker

type Api struct {
	Port         int
	LoadBalancer *LoadBalancer
}

type LoadBalancer struct {
	Port           int
	Workers        Workers
	WorkerPosition map[string]int
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Choose worker based on host
	workers, ok := lb.Workers[r.Host]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found.\n")
	}
	// Get data from worker
	resp, err := http.Get(fmt.Sprintf("http://%s:%d", lb.Workers[r.Host][lb.WorkerPosition[r.Host]].Address, workers[lb.WorkerPosition[r.Host]].Port))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error.\n")
	}
	defer resp.Body.Close()
	go func() {
		lb.WorkerPosition[r.Host] += 1
		lb.WorkerPosition[r.Host] = int(math.Mod(float64(lb.WorkerPosition[r.Host]), float64(len(lb.Workers[r.Host]))))
	}()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error.\n")
	}
	// Set response IP to loadbalancer IP
	// Return to requester
	fmt.Fprintf(w, string(bodyBytes))
}

func (lb *LoadBalancer) Start() {
	//TODO: make port configurable
	log.Fatal(http.ListenAndServe(":8081", lb))
}

func (a *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//TODO: prevent adding the same host twice
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var worker Worker
	err := decoder.Decode(&worker)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Invalid post data")
		return
	}
	a.LoadBalancer.Workers[worker.Host] = append(a.LoadBalancer.Workers[worker.Host], worker)
	fmt.Fprintf(w, "added worker %s to workers for host %s. There are now %d workers for %s\n", worker.Address, worker.Host, len(a.LoadBalancer.Workers[worker.Host]), worker.Host)
}

func (a *Api) Start() {
	//TODO: make port configurable
	log.Fatal(http.ListenAndServe(":8080", a))
}

func Start() {
	workers := make(map[string][]Worker)
	workerPositions := make(map[string]int)
	loadbalancer := LoadBalancer{8081, workers, workerPositions}
	api := Api{8080, &loadbalancer}
	go api.Start()
	loadbalancer.Start()
}
