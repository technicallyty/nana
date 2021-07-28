package main

import "errors"

func Hodor() error {
	err := errors.New("hi!")
	return err
}
