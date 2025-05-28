package main

// the flags for cgo below are a workaround for fixing some includes that
// seem to not be sent properly with nix. Hash for the nix store acquired
// by running `which tinygo`

// #cgo CFLAGS: -I /nix/store/73zik9gpf3n9vr699nyn77abk21v32kk-tinygo-0.31.2/share/tinygo/lib/wasi-libc/sysroot/include/
// #include <stdlib.h>
import "C"

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
