package main

import (
	"github.com/sxueck/kube-trash/cmd"
	"log"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}
