// To use this locally (SDK):
// 1. Run: gcloud beta auth application-default login
// 2. Login
// 3. Run: dev_appserver.py .

// localhost:8080/imagesFromAddress?address=1600+Amphitheatre+Parkway,+Mountain+View,+CA

package webservice

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	//"strconv"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
)

var DEBUG bool = false

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Geometry struct {
	Location Location `json:"location"`
}

type Results struct {
	Geometry Geometry `json:"geometry"`
}

type outer struct {
	Results []Results `json:"results"`
}

type data struct {
	ctx context.Context
}

func (d *data) doGet(url string) (latF float64, lngF float64, err error) {

	var res *http.Response
	if DEBUG {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return latF, lngF, err
		}
		client := http.DefaultClient
		res, err = client.Do(req)
		if err != nil {
			return latF, lngF, err
		}
	} else {
		client := urlfetch.Client(d.ctx)
		res, err = client.Get(url)
		if err != nil {
			return latF, lngF, err
		}
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return latF, lngF, err
	}
	log.Printf("Body is %s", b)

	var o outer
	err = json.Unmarshal([]byte(b), &o)

	loc := o.Results[0].Geometry.Location
	latF = loc.Lat
	lngF = loc.Lng

	{
		// var result map[string]interface{}
		// json.Unmarshal([]byte(b), &result)
		// location := result["results"].(interface{}).([]interface{})[0].(map[string]interface{})["geometry"].(map[string]interface{})["location"].(map[string]interface{})
		// for key, value := range location {
		// 	if key == "lat" {
		// 		lat := value.(string)
		// 		latF, err = strconv.ParseFloat(lat, 64)
		// 		if err != nil {
		// 			return
		// 		}
		// 	} else {
		// 		lng := value.(string)
		// 		lngF, err = strconv.ParseFloat(lng, 64)
		// 		if err != nil {
		// 			return
		// 		}
		// 	}
		// }
	}

	return
}

func imagesFromAddressHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("in imagesHandler")

	switch r.Method {
	case "GET":
		r.ParseForm()
		address := r.Form.Get("address")

		log.Printf("address is %s", address)

		geocodeUrl := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%v&key=%v", url.QueryEscape(address), geocodeApiKey)
		log.Printf("geocodeUrl is %s", geocodeUrl)

		ctx := appengine.NewContext(r)
		d := data{ctx: ctx}
		latF, lngF, _ := d.doGet(geocodeUrl)

		log.Printf("lat is %f and lng is %f", latF, lngF)

		rows, err := query(ctx, proj, latF, lngF)
		if err != nil {
			log.Fatal(err)
		}
		result, err := getResults(rows)
		if err != nil {
			log.Fatal(err)
		}

		var b [][]string
		for _, val := range result {
			img, _ := listImageFiles(ctx, val.URL)
			b = append(b, img)

		}

		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		err = enc.Encode(b)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
	}
}

func query(ctx context.Context, proj string, latF, lngF float64) (*bigquery.RowIterator, error) {
	client, err := bigquery.NewClient(ctx, proj)
	if err != nil {
		return nil, err
	}

	query := client.Query(
		`SELECT CONCAT(BASE_URL, '/GRANULE/', GRANULE_ID, '/IMG_DATA') AS URL FROM ` + "`bigquery-public-data.cloud_storage_geo_index.sentinel_2_index`" +
			`WHERE SOUTH_LAT < @LAT AND @LAT < NORTH_LAT AND WEST_LON < @LNG AND @LNG < EAST_LON 
			ORDER BY SENSING_TIME DESC
			LIMIT 1`)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "LAT", Value: latF},
		{Name: "LNG", Value: lngF},
	}

	return query.Read(ctx)
}

type sentinelData struct {
	URL string `bigquery:"url"`
}

func getResults(iter *bigquery.RowIterator) (result []sentinelData, err error) {
	for {
		var row sentinelData
		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		result = append(result, row)
	}
	return
}

func listImageFiles(ctx context.Context, imgDataUrl string) ([]string, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	u, err := url.Parse(imgDataUrl)
	if err != nil {
		return nil, err
	}

	bucket := client.Bucket(u.Host)

	prefix := strings.TrimLeft(u.Path, "/")
	log.Printf("Querying af %v", prefix)

	query := &storage.Query{Prefix: prefix}
	it := bucket.Objects(ctx, query)

	var images []string
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.Contains(obj.Name, "B02.jp2") || strings.Contains(obj.Name, "B03.jp2") || strings.Contains(obj.Name, "B04.jp2") {
			images = append(images, fmt.Sprintf("%v/%v", "gs://gcp-public-data-sentinel-2", obj.Name))

		}
	}
	return images, nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/imagesFromAddress", imagesFromAddressHandler)

	geocodeApiKey = os.Getenv("GEOCODE_API_KEY")
	if geocodeApiKey == "" {
		fmt.Println("GEOCODE_API_KEY environment variable must be set.")
		os.Exit(1)
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

var proj string
var geocodeApiKey string

func init() {
	log.Println("in init")
	proj = os.Getenv("GOOGLE_CLOUD_PROJECT") // environment variables are set in the yaml file
	if proj == "" {
		fmt.Println("GOOGLE_CLOUD_PROJECT environment variable must be set.")
		os.Exit(1)
	}

	geocodeApiKey = os.Getenv("GEOCODE_API_KEY")
	if geocodeApiKey == "" {
		fmt.Println("GEOCODE_API_KEY environment variable must be set.")
		os.Exit(1)
	}

	http.HandleFunc("/imagesFromAddress", imagesFromAddressHandler)
}
