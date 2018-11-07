Design and implement in Go a webservice, to be deployed on Google Cloud, that accepts GET requests containing the location of a point in space (latitude, longitude) as query parameters, e.g.: http://your.app.com/images?lat=37.4224764&lng=-122.0842499
and returns a JSON array containing the path to the images of the Red, Green and Blue bands for that location, e.g.: [“...2A_OPER_MSI_L1C_TL_EPA__20170529T193446_A002407_T33UUP_B01.jp2”,”...2A_OPER_MSI_L1C_TL_EPA__20170529T193446_A002407_T33UUP_B02.jp2”]


Noter:

https://bigquery.cloud.google.com/results/windy-renderer-215510:US.bquijob_21e46fb7_166ed63f55e?pli=1

SELECT base_url + '/GRANULE/' + granule_id + '/IMG_DATA' FROM [bigquery-public-data:cloud_storage_geo_index.sentinel_2_index] 

where south_lat < 37.4224764 and 37.4224764 < north_lat and west_lon < -122.0842499 and -122.0842499 < east_lon 

LIMIT 1000

--https://console.cloud.google.com/storage/browser/gcp-public-data-sentinel-2/tiles/10/S/EG/S2A_MSIL1C_20171206T190341_N0206_R113_T10SEG_20171206T202829.SAFE/GRANULE/L1C_T10SEG_A012837_20171206T190343/IMG_DATA
--tiles/10/S/EG/S2A_MSIL1C_20171206T190341_N0206_R113_T10SEG_20171206T202829.SAFE/GRANULE/L1C_T10SEG_A012837_20171206T190343/IMG_DATA