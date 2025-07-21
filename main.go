package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/nathnael-desta/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type errorReturn struct {
	Error string `json:"error"`
}

type returnVals struct {
	id        uuid.UUID     `json:"id"`
	createdAt time.Time     `json:"created_at"`
	updatedAt time.Time     `json:"updated_at"`
	body      string        `json:"body"`
	userID    uuid.NullUUID `json:"user_id"`
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

func (cfg *apiConfig) middlewareMetricsIncrease(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)

		next.ServeHTTP(w, r)
	})
}

func respondWithJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	log.Println(body)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("failed to write JSON response: %s", err))
	}
}

func respondWithError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	log.Printf("%s", err)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	params := parameters{}

	if OK := json.NewDecoder(r.Body).Decode(&params); OK != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "Couldn't decode request body"})
		return
	}

	if exists, err := cfg.dbQueries.EmailExists(r.Context(), params.Email); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to query %s", err))
		return
	} else if exists == 1 {
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("email already exists"))
		return
	}

	user, err := cfg.dbQueries.CreateUser(r.Context(), params.Email)

	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "faild to query"})
		return
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, fmt.Errorf("403 forbidden"))
		return
	}

	if err := cfg.dbQueries.Reset(r.Context()); err != nil {
		respondWithJSON(w, http.StatusInternalServerError, errorReturn{Error: "failed to query"})
	}

	// var empty interface{}
	w.WriteHeader(http.StatusNoContent)

	// respondWithJSON(w, http.StatusNoContent, empty)
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string `json:"body"`
		UserID string `json:"user_id"`
	}

	params := parameters{}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: fmt.Sprintf("Couldn't decode request body: %s", err)})
		return
	}

	if len(params.Body) > 140 {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "Chrip is too long"})
		return
	}

	replacedString, _ := replaceProfane(params.Body)
	params.Body = replacedString

	userUUID, err := uuid.Parse(params.UserID)

	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "invalid uuid"})
		return
	}

	chirpParams := database.CreateChirpParams{
		Body:   params.Body,
		UserID: uuid.NullUUID{UUID: userUUID, Valid: true},
	}

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), chirpParams)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("couldn't create chirp: %s", err))
		return
	}

	returnChirp := returnVals{
		id:        chirp.ID,
		createdAt: chirp.CreatedAt,
		updatedAt: chirp.UpdatedAt,
		body:      chirp.Body,
		userID:    chirp.UserID,
	}
	respondWithJSON(w, http.StatusCreated, returnChirp)
}

func replaceProfane(s string) (string, bool) {
	var profane = []string{"kerfuffle", "sharbert", "fornax"}
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

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, errorReturn{Error: "query faild"})
	}

	resp := make([]returnVals, 0, len(chirps))

	for _, v := range chirps {
		resp = append(resp, returnVals{
			id:        v.ID,
			createdAt: v.CreatedAt,
			updatedAt: v.UpdatedAt,
			body:      v.Body,
			userID:    v.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("id not given"))
		return
	}
	chirpId, err := uuid.Parse(parts[len(parts)-1])

	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Errorf("incorrect id format"))
		return
	}

	chirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpId)

	if err != nil {
		respondWithJSON(w, http.StatusNotFound, errorReturn{Error: "query faild"})
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)

}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("couldn't open database")
	}
	dbQueries := database.New(db)

	const filepathRoot = "."
	const port = "8080"
	myApiConfig := apiConfig{
		fileserverHits: atomic.Int32{},
		dbQueries:      dbQueries,
		platform:       os.Getenv("PLATFORM"),
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
	mux.HandleFunc("POST /api/users", myApiConfig.createUser)
	mux.HandleFunc("POST /api/chirps", myApiConfig.createChirp)
	mux.HandleFunc("GET /api/chirps", myApiConfig.getAllChirps)
	mux.HandleFunc("GET /api/chirps/{chirpid}", myApiConfig.getChirp)

	myServer := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	myServer.ListenAndServe()

}
