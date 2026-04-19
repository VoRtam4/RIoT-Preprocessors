package sharedUtils

import (
	"bytes"
	"log"

	"github.com/davecgh/go-spew/spew"
)

func Dump(a ...any) {
	buffer := new(bytes.Buffer)
	spew.Fdump(buffer, a)
	log.Println(buffer.String())
}
