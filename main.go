package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) returnHits(w http.ResponseWriter, r *http.Request) {
	 fmt.Fprintf(w,"Hits: %v\n", cfg.fileserverHits.Load())
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	 cfg.fileserverHits.Swap(0)
}

func (cfg *apiConfig) middlewareMetricsIncrease(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r * http.Request) {
		cfg.fileserverHits.Add(1)

		next.ServeHTTP(w,r)
	})
}




func main() {
	const filepathRoot = "."
	const port = "8080"
	myApiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
	}


	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", myApiConfig.middlewareMetricsIncrease(http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("/healthz", func (w http.ResponseWriter,r *http.Request) {
		w.Header().Set("Content-type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/metrics", myApiConfig.returnHits)
	mux.HandleFunc("/reset", myApiConfig.reset)

	myServer := http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	myServer.ListenAndServe()

}