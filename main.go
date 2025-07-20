package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) returnHits(w http.ResponseWriter, r *http.Request) {
	html := fmt.Sprintf(`
	 <html>
  		<body>
    		<h1>Welcome, Chirpy Admin</h1>
    		<p>Chirpy has been visited %d times!</p>
  		</body>
	</html>`, cfg.fileserverHits.Load())
	w.Header().Set("Content-type", "text/html;")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Swap(0)
}

func (cfg *apiConfig) middlewareMetricsIncrease(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)

		next.ServeHTTP(w, r)
	})
}

func validate_chirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type errorReturn struct {
		Error string `json:"error"`
	}

	type returnVals struct {
		Valid bool `json:"valid"`
	}


	params := parameters{}
	

	if OK := json.NewDecoder(r.Body).Decode(&params); OK != nil {
		respondJSON(w, http.StatusBadRequest, errorReturn{Error: "Couldn't decode request body"})
		return
	}

	if len(params.Body) > 140 {
		respondJSON(w, http.StatusBadRequest, errorReturn{Error: "Chrip is too long"})
		return
	}

	respondJSON(w, http.StatusOK, returnVals{Valid: true})
}

func respondJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("Failed to write JSON response: %s", err)
	}
}

func main() {
	const filepathRoot = "."
	const port = "8080"
	myApiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", myApiConfig.middlewareMetricsIncrease(http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /admin/metrics", myApiConfig.returnHits)
	mux.HandleFunc("POST /admin/reset", myApiConfig.reset)
	mux.HandleFunc("POST /api/validate_chirp", validate_chirp)

	myServer := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	myServer.ListenAndServe()

}
