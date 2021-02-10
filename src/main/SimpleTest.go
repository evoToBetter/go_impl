package main

import (
	"log"
	"time"
)

func main() {
	testTime()
}

func testTime() {
	now := time.Now()
	log.Printf("current time: %v", now)
	log.Printf("current format time: %v", now.Format("2006-01-02 15:04:05"))
	log.Printf("current year: %v", now.Year())
	log.Printf("current month: %v", int(now.Month()))
	log.Printf("current day: %v", now.Day())
	log.Printf("current weekday: %v", int(now.Weekday()))
	log.Printf("current hour: %v", now.Hour())
	log.Printf("current minutes: %v", now.Minute())
	log.Printf("current second: %v", now.Second())
	time.Sleep(time.Second)
	nextNow := time.Now()
	log.Printf("current duration: %v", nextNow.Sub(now))
}
