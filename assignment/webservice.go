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
		value := r.Form.Get("rankByValue")

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

		if len(value) == 6 {
			b, err = d.rankByColor(value, b, false)
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

func (d *data) rankByBand(band string, imagePaths [][]string) ([][]string, error) {
	log.Printf("Ranking based on band %s", band)

	var value string
	switch band {
	case "B02":
		value = "0000ff"
	case "B03":
		value = "00ff00"
	case "B04":
		value = "ff0000"
	default:
		return nil, fmt.Errorf("band %v not allowed", band)
	}

	return d.rankByColor(value, imagePaths, true)
}

type distImg struct {
	distance   float64
	imagePaths []string
}

func (d *data) rankByColor(value string, imagePaths [][]string, byBand bool) ([][]string, error) {
	log.Printf("Ranking based on value %s", value)

	r, g, b, err := getRGBFromHex(value)
	if err != nil {
		return nil, err
	}
	target := [3]float64{getLuminosity(r, 0, 0), getLuminosity(0, g, 0), getLuminosity(0, 0, b)}

	log.Printf("target is %v", target)

	getAvg := func(urlStr string) (float64, error) {
		u, err := url.Parse(urlStr)
		if err != nil {
			return 0, err
		}

		path := strings.TrimLeft(u.Path, "/")
		a, err := d.averageColor(path)
		if err != nil {
			return 0, err
		}
		return a, nil
	}

	var distImgList []distImg
	for i := 0; i < len(imagePaths); i++ {
		row := imagePaths[i]
		dist := 0.0
		var rAvg, gAvg, bAvg float64
		for j := 0; j < len(row); j++ {
			if strings.Contains(row[j], band4File) && (target[0] > 0.0 || !byBand) {
				rAvg, _ = getAvg(row[j])
			}
			if strings.Contains(row[j], band3File) && (target[1] > 0.0 || !byBand) {
				gAvg, _ = getAvg(row[j])
			}
			if strings.Contains(row[j], band2File) && (target[2] > 0.0 || !byBand) {
				bAvg, _ = getAvg(row[j])
			}
		}
		dist = distance(target, [3]float64{rAvg, gAvg, bAvg})
		log.Printf("dist is %v", dist)
		distImgList = append(distImgList, distImg{distance: dist, imagePaths: row})
	}

	sort.Slice(distImgList, func(i, j int) bool {
		return distImgList[i].distance < distImgList[j].distance
	})

	// log.Printf("imagePaths is %v", imagePaths)

	var result [][]string
	for _, val := range distImgList {
		result = append(result, val.imagePaths)
	}

	// log.Printf("result is %v", result)

	return result, nil
}

func getRGBFromHex(value string) (int, int, int, error) {
	a := []rune(value)

	r, err := getInt(string(a[0:2]))
	if err != nil {
		return 0, 0, 0, err
	}
	g, err := getInt(string(a[2:4]))
	if err != nil {
		return 0, 0, 0, err
	}
	b, err := getInt(string(a[4:6]))
	if err != nil {
		return 0, 0, 0, err
	}

	return r, g, b, nil
}

func getInt(s string) (int, error) {
	n, err := strconv.ParseInt(s, 16, 0)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func getLuminosity(r, g, b int) float64 {
	rgbLum := 0.21*float64(r) + 0.72*float64(g) + 0.07*float64(b)
	return 10000.0 / 256 * rgbLum // One of the operands must be a floating-point constant for the result to a floating-point constant (https://stackoverflow.com/a/32815507)
}

// distance calculates the Eucleadian distance between 2 points
func distance(p1 [3]float64, p2 [3]float64) float64 {
	return math.Sqrt(sq(p2[0]-p1[0]) + sq(p2[1]-p1[1]) + sq(p2[2]-p1[2]))
}

func sq(n float64) float64 {
	return n * n
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
