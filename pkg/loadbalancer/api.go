package loadbalancer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"sync"
)

type Worker struct {
	Host            string
	Address         string
	Port            int
	HealthcheckPath string
}

type Workers struct {
	Items    map[string][]Worker
	Position map[string]int
	Mutex    map[string]*sync.Mutex
}

type Api struct {
	Port         int
	LoadBalancer *LoadBalancer
}

type LoadBalancer struct {
	Workers
	Port int
}

func (w *Workers) Get(key string) []Worker {
	return w.Items[key]
}

func (w *Workers) Set(host string, value Worker) {
	w.Mutex[host].Lock()
	defer w.Mutex[host].Unlock()
	w.Items[host] = append(w.Items[host], value)
}

func (w *Workers) GetPosition(host string) int {
	w.Mutex[host].Lock()
	defer w.Mutex[host].Unlock()
	return w.Position[host]
}

func (w *Workers) Inc(host string) {
	w.Mutex[host].Lock()
	defer w.Mutex[host].Unlock()
	w.Position[host] = int(math.Mod(float64(w.Position[host])+1, float64(len(w.Get(host)))))
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Choose worker based on host
	workers := lb.Workers.Get(r.Host)
	if len(workers) == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Not found.\n")
	}
	// Get data from worker
	position := lb.GetPosition(r.Host)
	resp, err := http.Get(fmt.Sprintf("http://%s:%d", workers[position].Address, workers[position].Port))
	go func() {
		lb.Inc(r.Host)
	}()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error.\n")
		return
	}

	defer resp.Body.Close()

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
	if a.LoadBalancer.Mutex[worker.Host] == nil {
		a.LoadBalancer.Mutex[worker.Host] = &sync.Mutex{}
	}
	a.LoadBalancer.Set(worker.Host, worker)
	fmt.Fprintf(w, "added worker %s to workers for host %s. There are now %d workers for %s\n", worker.Address, worker.Host, len(a.LoadBalancer.Get(worker.Host)), worker.Host)
}

func (a *Api) Start() {
	//TODO: make port configurable
	log.Fatal(http.ListenAndServe(":8080", a))
}

func Start() {
	workers := make(map[string][]Worker)
	workerPositions := make(map[string]int)
	mutex := make(map[string]*sync.Mutex)
	loadbalancer := LoadBalancer{Workers{workers, workerPositions, mutex}, 8081}
	api := Api{8080, &loadbalancer}
	go api.Start()
	loadbalancer.Start()
}
