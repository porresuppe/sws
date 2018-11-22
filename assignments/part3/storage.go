package webservice

import (
	"cloud.google.com/go/storage"
	"fmt"
	"google.golang.org/api/iterator"
	"log"
	"net/url"
	"strings"
)

func (d *data) listImageFiles(imgDataUrl string) ([]string, error) {
	client, err := storage.NewClient(d.ctx)
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
	it := bucket.Objects(d.ctx, query)

	var images []string
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.Contains(obj.Name, band2File) || strings.Contains(obj.Name, band3File) || strings.Contains(obj.Name, band4File) {
			images = append(images, fmt.Sprintf("%v/%v", "gs://gcp-public-data-sentinel-2", obj.Name))

		}
	}
	return images, nil
}
