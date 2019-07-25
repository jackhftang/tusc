package util

import (
	"fmt"
	"os"
)

func ExitWithMessages(msg ...interface{}) {
	for _, m := range msg {
		fmt.Fprintln(os.Stderr, m)
	}
	//fmt.Fprintln(os.Stderr, msg...)
	os.Exit(1)
}
