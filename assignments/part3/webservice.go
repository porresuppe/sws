package webservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/appengine"
)

type data struct {
	ctx context.Context
}

var band2File = "B02.jp2" // Blue band
var band3File = "B03.jp2" // Green band
var band4File = "B04.jp2" // Red band

func imagesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("in imagesHandler")

	switch r.Method {
	case "GET":
		r.ParseForm()
		lat := r.Form.Get("lat")
		lng := r.Form.Get("lng")
		band := r.Form.Get("rankByBand")

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

		ctx := appengine.NewContext(r)
		d := data{ctx: ctx}
		rows, err := d.query(proj, latF, lngF)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result, err := getResults(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var b [][]string
		for _, val := range result {
			img, err := d.listImageFiles(val.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b = append(b, img)
		}

		if band != "" {
			b, err = d.rankByBand(band, b)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		err = enc.Encode(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
	}
}

func imagesFromAddressHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("in imagesHandler")

	switch r.Method {
	case "GET":
		r.ParseForm()
		address := r.Form.Get("address")

		log.Printf("address is %s", address)

		ctx := appengine.NewContext(r)
		d := data{ctx: ctx}
		latF, lngF, _ := d.getLocation(address, geocodeAPIKey)

		log.Printf("lat is %f and lng is %f", latF, lngF)

		rows, err := d.query(proj, latF, lngF)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result, err := getResults(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var b [][]string
		for _, val := range result {
			img, err := d.listImageFiles(val.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			b = append(b, img)
		}

		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		err = enc.Encode(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
	}
}

type avgColor struct {
	averageColor int
	imagePaths   []string
}

func (d *data) rankByBand(band string, imagePaths [][]string) ([][]string, error) {
	log.Printf("Ranking based on band %s", band)

	var bFile string
	var target int
	switch band {
	case "B02":
		bFile = band2File
		target = getLuminosity(0, 0, 255)
	case "B03":
		bFile = band3File
		target = getLuminosity(0, 255, 0)
	case "B04":
		bFile = band4File
		target = getLuminosity(255, 0, 0)
	default:
		return nil, fmt.Errorf("band %v not allowed", band)
	}

	log.Printf("bFile is %v, target is %d", bFile, target)

	var avgColors []avgColor
	for i := 0; i < len(imagePaths); i++ {
		row := imagePaths[i]
		for j := 0; j < len(row); j++ {
			if strings.Contains(row[j], bFile) {
				u, err := url.Parse(row[j])
				if err != nil {
					return nil, err
				}

				path := strings.TrimLeft(u.Path, "/")
				a, err := d.averageColor(path)
				if err != nil {
					return nil, err
				}

				avgColors = append(avgColors, avgColor{averageColor: a, imagePaths: row})
			}
		}
	}

	sort.Slice(avgColors, func(i, j int) bool {
		return math.Abs(float64(target-avgColors[i].averageColor)) < math.Abs(float64(target-avgColors[j].averageColor))
	})

	var result [][]string
	for _, val := range avgColors {
		result = append(result, val.imagePaths)
	}
	return result, nil
}

func getLuminosity(r, g, b int) int {
	rgbLum := 0.21*float64(r) + 0.72*float64(g) + 0.07*float64(b)
	return int(round(10000.0/256*rgbLum, 0.5, 0)) // One of the operands must be a floating-point constant for the result to a floating-point constant (https://stackoverflow.com/a/32815507)
}

func round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

var proj string
var geocodeAPIKey string

func init() {
	log.Println("in init")
	proj = os.Getenv("GOOGLE_CLOUD_PROJECT") // environment variables are set in the yaml file
	if proj == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT environment variable must be set.")
		return
	}

	geocodeAPIKey = os.Getenv("GEOCODE_API_KEY")
	if geocodeAPIKey == "" {
		log.Fatal("GEOCODE_API_KEY environment variable must be set.")
		return
	}

	http.HandleFunc("/images", imagesHandler)
	http.HandleFunc("/imagesFromAddress", imagesFromAddressHandler)
}
