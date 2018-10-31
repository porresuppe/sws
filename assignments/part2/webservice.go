package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/im7mortal/UTM"
)

func imagesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		r.ParseForm()
		lat := r.Form.Get("lat")
		lng := r.Form.Get("lng")

		log.Printf("lat is %s and lng is %s", lat, lng)

		latF, err := strconv.ParseFloat(lat, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lngF, err := strconv.ParseFloat(lng, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, _, zoneNumber, zoneLetter, err := UTM.FromLatLon(latF, lngF, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("ZoneNumber: %d; ZoneLetter: %s;", zoneNumber, zoneLetter)

	default:
		http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/images", imagesHandler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
