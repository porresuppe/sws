// To use this locally (SDK):
// 1. Run: gcloud beta auth application-default login
// 2. Login
// 3. Run: dev_appserver.py .

package webservice

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
)

func imagesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("in imagesHandler")

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

		ctx := appengine.NewContext(r)
		rows, err := query(ctx, proj, latF, lngF)
		if err != nil {
			log.Fatal(err)
		}
		result, err := getResults(rows)
		if err != nil {
			log.Fatal(err)
		}

		for _, val := range result {
			listImageFiles(ctx, val.URL)
		}
	default:
		http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
	}
}

// TODO: use latF, lngF
func query(ctx context.Context, proj string, latF, lngF float64) (*bigquery.RowIterator, error) {
	client, err := bigquery.NewClient(ctx, proj)
	if err != nil {
		return nil, err
	}

	query := client.Query(
		`SELECT CONCAT(base_url, '/GRANULE/', granule_id, '/IMG_DATA') as url FROM ` + "`bigquery-public-data.cloud_storage_geo_index.sentinel_2_index`" +
			`where south_lat < 37.4224764 and 37.4224764 < north_lat and west_lon < -122.0842499 and -122.0842499 < east_lon 
			LIMIT 3`)

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

func listImageFiles(ctx context.Context, imgDataUrl string) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Panicf("failed to create client: %v", err)
		return
	}
	defer client.Close()

	u, err := url.Parse(imgDataUrl)
	if err != nil {
		log.Fatal(err)
	}

	bucket := client.Bucket(u.Host)

	exists, err := bucket.Attrs(ctx)
	if err != nil {
		log.Fatalf("Message: %v", err)
	}
	log.Println(exists)

	prefix := strings.TrimLeft(u.Path, "/")
	log.Printf("Querying af %v", prefix)
	query := &storage.Query{Prefix: prefix}
	it := bucket.Objects(ctx, query)
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Panicf("unable to list bucket %q: %v", imgDataUrl, err)
			return
		}
		log.Printf("(bucket: %v, name: %v", obj.Bucket, obj.Name)
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

var proj string

func init() {
	log.Println("in init")
	proj = os.Getenv("GOOGLE_CLOUD_PROJECT") // environment variables are set in the yaml file
	if proj == "" {
		fmt.Println("GOOGLE_CLOUD_PROJECT environment variable must be set.")
		os.Exit(1)
	}

	http.HandleFunc("/images", imagesHandler)
}
