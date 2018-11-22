package webservice

import (
	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

func (d *data) query(proj string, latF, lngF float64) (*bigquery.RowIterator, error) {
	client, err := bigquery.NewClient(d.ctx, proj)
	if err != nil {
		return nil, err
	}

	limit := 3

	query := client.Query(
		`SELECT CONCAT(BASE_URL, '/GRANULE/', GRANULE_ID, '/IMG_DATA') AS URL FROM ` + "`bigquery-public-data.cloud_storage_geo_index.sentinel_2_index`" +
			`WHERE SOUTH_LAT < @LAT AND @LAT < NORTH_LAT AND WEST_LON < @LNG AND @LNG < EAST_LON 
			ORDER BY SENSING_TIME DESC
			LIMIT @LIMIT`)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "LAT", Value: latF},
		{Name: "LNG", Value: lngF},
		{Name: "LIMIT", Value: limit},
	}

	return query.Read(d.ctx)
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
