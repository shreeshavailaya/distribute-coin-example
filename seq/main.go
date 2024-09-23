package main

import (
	"log"
	"net/http"
)

func main() {
	for i := 0; i < 100; i++ {
		resp, err := http.Get("http://localhost:8080/distribute?address=0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("response %v", resp)
	}
}
