package main

import (
	"log"

	"github.com/alecthomas/kong"
)

var cli struct {
	Init    InitCmd    `cmd help:"Initialize a github-secrets project"`
	Update  UpdateCmd  `cmd help:"Updates secrets for a given project"`
	Extract ExtractCmd `cmd help:"Extracts secrets for a given project"`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}
}
