package main

import "fmt"

const (
	bro = "hi"
	dude = 1
)

type Yolo struct {
	duder int
}

func (y *Yolo) omg() int {
	// wow comments!
	return y.duder
}

// this is a comment
func Hi() {
	fmt.Println("hello")
}
