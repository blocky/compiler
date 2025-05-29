package main

import (
	"fmt"

	"golang.org/x/crypto/sha3"
)

func main() {
	s := "hello world"
	hash := sha3.Sum512([]byte(s))
	fmt.Println("Hello World")
	fmt.Printf("Hello Hash: %x\n", hash)
}
