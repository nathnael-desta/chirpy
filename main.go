package main

import (
	"context"
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
	"github.com/nathnael-desta/chirpy/internal/auth"
	"github.com/nathnael-desta/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	tokenSecret    string
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
	Id        uuid.UUID     `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Body      string        `json:"body"`
	UserID    uuid.NullUUID `json:"user_id"`
}

type userReturn struct {
	Id           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type userParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateChirpParams struct {
	Body   string `json:"body"`
	UserID string `json:"user_id"`
}

type RefreshTokenReturn struct {
	Token string `json:"token"`
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
	// log.Println(body)
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

	params := userParams{}

	if OK := json.NewDecoder(r.Body).Decode(&params); OK != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: fmt.Sprintf("Couldn't decode request body %s", OK)})
		return
	}

	if _, err := cfg.dbQueries.EmailExists(r.Context(), params.Email); err != sql.ErrNoRows {
		if err == nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("email already exists"))
			return
		} else {
			respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to query for email %s", err))
			return
		}
	}

	hashedPassword, err := auth.HashPassword(params.Password)

	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, errorReturn{Error: "password hashing failed"})
		return
	}

	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: hashedPassword})

	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "faild to query for create user"})
		return
	}

	token, err := getToken(user.ID, cfg.tokenSecret)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	refreshToken, err := getRefreshToken(r.Context(), cfg, user.ID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	returnVals := userReturn{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken.Token,
	}

	respondWithJSON(w, http.StatusCreated, returnVals)
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
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err)
		return
	}

	if _, err := auth.ValidateJWT(token, cfg.tokenSecret); err != nil {
		respondWithError(w, http.StatusUnauthorized, err)
		return
	}

	params := CreateChirpParams{}

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
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
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
			Id:        v.ID,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
			Body:      v.Body,
			UserID:    v.UserID,
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

func (cfg *apiConfig) logIn(w http.ResponseWriter, r *http.Request) {
	params := userParams{}

	if OK := json.NewDecoder(r.Body).Decode(&params); OK != nil {
		respondWithJSON(w, http.StatusBadRequest, errorReturn{Error: "Couldn't decode request body"})
		return
	}

	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), params.Email)

	if err != nil {
		respondWithJSON(w, http.StatusNotFound, errorReturn{Error: "Incorrect email or password"})
		return
	}

	if err := auth.CheckPasswordHash(params.Password, user.HashedPassword); err != nil {
		respondWithJSON(w, http.StatusUnauthorized, errorReturn{Error: "Incorrect email or password"})
		return
	}

	token, err := getToken(user.ID, cfg.tokenSecret)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	refreshToken, err := getRefreshToken(r.Context(), cfg, user.ID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	returnVals := userReturn{
		Id:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken.Token,
	}

	respondWithJSON(w, http.StatusOK, returnVals)

}

func getRefreshToken(context context.Context, cfg *apiConfig, userID uuid.UUID) (database.RefreshToken, error) {
	refreshTokenCode, err := auth.MakeRefreshToken()

	if err != nil {
		return database.RefreshToken{}, err
	}

	refreshTokenParams := database.CreateRefreshTokenParams{
		Token: refreshTokenCode,
		UserID: uuid.NullUUID{
			UUID:  userID,
			Valid: true,
		},
		ExpiresAt: time.Now().Add(time.Duration(60*60*24*60) * time.Second),
	}

	refreshToken, err := cfg.dbQueries.CreateRefreshToken(context, refreshTokenParams)

	if err != nil {
		return database.RefreshToken{}, err
	}
	return refreshToken, nil
}

func getToken(userID uuid.UUID, tokenSecret string) (string, error) {
	token, err := auth.MakeJWT(userID, tokenSecret, time.Duration(3600)*time.Second)

	if err != nil {
		return "", err
	}
	return token, nil
}

func checkRefreshToken(cfg *apiConfig, ctx context.Context, token string) (database.RefreshToken, error) {
	refreshToken, err := cfg.dbQueries.GetRefreshToken(ctx, token)
	if err != nil {
		return database.RefreshToken{}, err
	}
	log.Println(refreshToken.ExpiresAt, "\n", time.Now(), "\n", refreshToken.ExpiresAt.Before(time.Now()), "////////////////////////////////")

	if refreshToken.ExpiresAt.Before(time.Now()) {
		// handle expired token, e.g., return an error
		return database.RefreshToken{}, fmt.Errorf("refresh token has expired")
	}
	if refreshToken.RevokedAt.Valid {
		// handle expired token, e.g., return an error
		return database.RefreshToken{}, fmt.Errorf("refresh token has been revoked")
	}
	return refreshToken, nil
}

func (cfg *apiConfig) refreshToken(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}
	refreshToken, err := checkRefreshToken(cfg, r.Context(), token)

	if err != nil {
		respondWithError(w, http.StatusUnauthorized, fmt.Errorf("refresh token failed: %v", err))
		return
	}

	newToken, err := auth.MakeJWT(refreshToken.UserID.UUID, cfg.tokenSecret, time.Duration(60*60)*time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJSON(w, http.StatusOK, RefreshTokenReturn{Token: newToken})
}

func (cfg *apiConfig) revokeRefresh(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if err := cfg.dbQueries.RevokeRefreshToken(r.Context(), token); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	var emptyReturn interface{}

	respondWithJSON(w, http.StatusNoContent, emptyReturn)

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
		tokenSecret:    os.Getenv("JWT_SECRET"),
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
	mux.HandleFunc("POST /api/login", myApiConfig.logIn)
	mux.HandleFunc("POST /api/refresh", myApiConfig.refreshToken)
	mux.HandleFunc("POST /api/revoke", myApiConfig.revokeRefresh)

	myServer := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	myServer.ListenAndServe()

}
