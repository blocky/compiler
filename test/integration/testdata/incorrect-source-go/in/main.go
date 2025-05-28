package main

// the flags for cgo below are a workaround for fixing some includes that
// seem to not be sent properly with nix. Hash for the nix store acquired
// by running `which tinygo`

// #cgo CFLAGS: -I /nix/store/73zik9gpf3n9vr699nyn77abk21v32kk-tinygo-0.31.2/share/tinygo/lib/wasi-libc/sysroot/include/
// #include <stdlib.h>
import "C"

import (
	"fmt"
)

func main() {
	fmt.Println("Hello World")
	this file should not compile
}
