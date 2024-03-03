package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type HealthCheckJSON struct {
	Status string `json:"status"`
}

func main() {
	resp, err := http.Get("http://127.0.0.1:8080/healthcheck")

	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Fatalln(err)
	}

	var healthcheck HealthCheckJSON
	err = json.Unmarshal(body, &healthcheck)
	if err != nil {
		log.Fatalln(err)
	}
	if healthcheck.Status != "ok" {
		log.Fatalln("status is not ok")
	}
	log.Println("status is ok")
}
