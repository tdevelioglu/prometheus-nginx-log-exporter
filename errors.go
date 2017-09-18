package main

import "fmt"

type requestParseError string

func (e requestParseError) Error() string {
	return fmt.Sprintf("%s: invalid request", e)
}
