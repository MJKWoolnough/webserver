package main

import "time"

type Event struct {
	Start, End  time.Time
	TreatmentID int
}

type Calendar struct {
}

func (c *Calendar) IsFree(e Event) bool {
	return false
}

func (c *Calendar) Set(e Event) error {
	return nil
}

func (c *Calendar) GetEvents(start, end time.Time) []Event {
	return nil
}
