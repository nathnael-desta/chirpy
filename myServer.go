package main
import (
	"net/http"
)


func main() {
	const filepathRoot = "."
	const port = "8080"

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(filepathRoot)))

	myServer := http.Server{
		Addr: ":" + port,
		Handler: mux,
	}
	myServer.ListenAndServe()

}