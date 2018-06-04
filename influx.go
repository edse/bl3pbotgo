package main

import (
	"fmt"
	"log"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

func GetClient() client.Client {
	// Create a new HTTPClient
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://influx:8086",
		Username: "root",
		Password: "root",
	})
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func Store(db string, pair string, tags map[string]string, fields map[string]interface{}, time time.Time) {
	fmt.Println("Storing into influx...")
	fmt.Println("db: ", db)
	fmt.Println("pair: ", pair)
	fmt.Println("tags: ", tags)
	fmt.Println("fields: ", fields)

	c := GetClient()
	defer c.Close()

	// Create a new point batch
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  db,
		Precision: "s",
	})
	if err != nil {
		log.Fatal(err)
	}

	pt, err := client.NewPoint(pair, tags, fields, time)
	if err != nil {
		log.Fatal(err)
	}
	bp.AddPoint(pt)

	// Write the batch
	if err := c.Write(bp); err != nil {
		log.Fatal(err)
	}

	// Close client resources
	if err := c.Close(); err != nil {
		log.Fatal(err)
	}
}

func Query(c client.Client, db string, cmd string) (res []client.Result) {
	q := client.Query{
		Command:  cmd,
		Database: db,
	}
	if response, err := c.Query(q); err == nil {
		if response.Error() != nil {
			return res
		}
		res = response.Results
	} else {
		return res
	}
	return res
}
