package webservice

import (
	"bytes"
	"encoding/json"
	"google.golang.org/appengine/urlfetch"
	"io/ioutil"
	"log"
)

type jp2Request struct {
	Path   string `json:"path"`
	Rlevel int    `json:"rlevel"`
}

type jp2Response struct {
	ImageData      [][]int `json:"img_data"`
	Shape          []int   `json:"shape"`
	TimeDownload   float64 `json:"time_download"`
	TimeProcessing float64 `json:"time_processing"`
}

func (d *data) averageColor(path string) (int, error) {
	reqBody, err := json.Marshal(jp2Request{Path: path, Rlevel: -1})
	if err != nil {
		return 0, err
	}

	url := "http://35.227.24.82/api/jp2"
	client := urlfetch.Client(d.ctx)
	response, err := client.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}
	//log.Printf("Body is %s", body)

	var jp2Res jp2Response
	err = json.Unmarshal([]byte(body), &jp2Res)

	total := 0
	for i := 0; i < len(jp2Res.ImageData); i++ {
		row := jp2Res.ImageData[i]
		for j := 0; j < len(row); j++ {
			total += row[j]
		}
	}
	average := total / (jp2Res.Shape[0] * jp2Res.Shape[1])

	log.Printf("average is %v", average)

	return average, nil
}
