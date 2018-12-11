package main

import (
	"encoding/json"
	"fmt"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"log"
	"net/url"
)

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

func (d *data) getLocation(address string, geocodeAPIKey string) (latF float64, lngF float64, err error) {
	geocodeURL := fmt.Sprintf("https://maps.googleapis.com/maps/api/geocode/json?address=%v&key=%v", url.QueryEscape(address), geocodeAPIKey)
	log.Printf("geocodeURL is %s", geocodeURL)

	client := urlfetch.Client(d.ctx)
	res, err := client.Get(geocodeURL)
	if err != nil {
		log.Printf("Error is: %s", err.Error())
		return latF, lngF, err
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
