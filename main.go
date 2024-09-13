package main

import (
	"github.com/sxueck/kube-record/cmd"
	"log"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}
