package main

import (
	"log"

	"github.com/warthog618/go-gpiocdev"
)

func main() {
	c, err := gpiocdev.NewChip("gpiochip0")
	if err != nil {
		log.Println(err)
	}
	defer c.Close()

	l, err := c.RequestLines([]int{19, 20, 21}, gpiocdev.AsOutput(1, 1, 1))
	if err != nil {
		log.Println("RequestLines:", err)
	}
	defer l.Close()

	rr := []int{0, 0, 0}

	l.Values(rr)
	log.Println("Init Values", rr)

	err1 := l.SetValues([]int{0, 1, 0})
	if err1 != nil {
		log.Println("SetValues:", err1)
	}
	l.Values(rr)
	log.Println("Set Values", rr)

	// this must be here for SetValues() to have effect
	c.RequestLine(16, gpiocdev.AsOutput(1))

	l.Values(rr)
	log.Println("Values", rr)
}
