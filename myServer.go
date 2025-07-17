package main
import (
	"net/http"
)




func main() {
	const filepathRoot = "."
	const port = "8080"

	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot))))
	mux.HandleFunc("/healthz", func (w http.ResponseWriter,r *http.Request) {
		w.Header().Set("Content-type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	myServer := http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	myServer.ListenAndServe()

}