package main

import "strings"

type task struct {
	pomodoros   int
	completed   int
	description string
}

func (t task) String() string {
	var sb strings.Builder
	sb.WriteString(t.PomoString())
	sb.WriteString(t.description)
	return sb.String()
}

func (t task) PomoString() string {
	var sb strings.Builder
	for i := 0; i < t.completed; i++ {
		sb.WriteString("⬢ ")
	}
	for i := 0; i < t.pomodoros-t.completed; i++ {
		sb.WriteString("⬡ ")
	}
	return sb.String()
}
