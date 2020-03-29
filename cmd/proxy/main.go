package main

import (
	"net/http"
	"os"
)

// get env var or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// get the port to listen on
func getListenAddress() string {
	port := getEnv("PORT", "8189")
	return ":" + port
}

func main() {
	// start the server
	http.HandleFunc("/hook", handleRequestAndRedirect)
	if err := http.ListenAndServe(getListenAddress(), nil); err != nil {
		panic(err)
	}
}
