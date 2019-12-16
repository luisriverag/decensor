package main

import (
	"log"
	"time"

	"gopkg.in/alexcesaro/statsd.v2"
)

// Seconds
const statsdReportingInterval = 60

func statsdLoop(s *statsd.Client) {
	for true {
		tagsCount, err := countTags()
		if err != nil {
			log.Print(err.Error())
		} else {
			log.Printf("Tags count: %d", tagsCount)
			s.Gauge("tags.count", tagsCount)
		}
		assetsCount, err := countAssets()
		if err != nil {
			log.Print(err.Error())
		} else {
			log.Printf("Assets count: %d", assetsCount)
			s.Gauge("assets.count", assetsCount)
		}
		time.Sleep(statsdReportingInterval * time.Second)
	}
}
