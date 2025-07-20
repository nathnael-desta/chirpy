package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
		Valid *bool `json:"valid,omitempty"`
		Cleaned_Body *string `json:"cleaned_body,omitempty"`
	}


	params := parameters{}


	if OK := json.NewDecoder(r.Body).Decode(&params); OK != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "Couldn't decode request body"})
		return
	}


	if len(params.Body) > 140 {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "Chrip is too long"})
		return
	}

	
	if replacedString, modified := replaceProfane(params.Body); modified {
		respondWithJSON(w, http.StatusOK, returnVals{Cleaned_Body: &replacedString})
		return
	}

	 validTrue := true
    respondWithJSON(w, http.StatusOK, returnVals{
        Valid: &validTrue,
    })
}


func replaceProfane(s string) (string, bool) {
	var profane = []string{"kerfuffle","sharbert", "fornax"}
	split_sentence := strings.Split(strings.ToLower(s), " ")
	modified := false
	for _, p := range profane {
		for i, w := range split_sentence {
			if w == p {
				split_sentence[i] = "****"
				modified = true
				break
			} 
		}
	}
	return strings.Join(split_sentence, " "), modified
}

func respondWithJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to write JSON response:", err)
	}
}

func respondWithError(w http.ResponseWriter, status int, msg string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	log.Printf("%v: %s",msg, err)
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
