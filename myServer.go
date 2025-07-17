package main
import (
	"net/http"
)


func main() {
	mux := http.NewServeMux()
	myServer := http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	myServer.ListenAndServe()

}