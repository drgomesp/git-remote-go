package main

import (
	"os"
	"path"

	"github.com/rs/zerolog/log"

	"github.com/drgomesp/git-remote-go/pkg/gitremotego"
	gitremote "github.com/drgomesp/git-remote-go/pkg/gitremotego-ipfs"
)

func init() {
	wdir, err := os.Getwd()
	if err != nil {
		log.Err(err).Send()
	}

	if err = os.Setenv(
		"GIT_DIR",
		path.Join(wdir),
	); err != nil {
		log.Err(err).Send()
	}
}

func main() {
	if os.Getenv("GIT_DIR") == "" {
		log.Fatal().Msg("missing repository path ($GIT_DIR)")
	}

	handler, err := gitremote.NewIpfsProtocol()
	if err != nil {
		log.Err(err).Send()
	}

	proto, err := gitremotego.NewProtocol("gitremotego", handler)
	if err != nil {
		log.Err(err).Send()
	}

	if err := proto.Run(os.Stdin, os.Stdout); err != nil {
		log.Err(err).Send()
	}
}
