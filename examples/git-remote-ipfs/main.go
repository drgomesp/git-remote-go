package main

import (
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog/log"

	"github.com/drgomesp/git-remote-go/pkg/gitremotego"
	gitremote "github.com/drgomesp/git-remote-go/pkg/gitremotego-ipfs"
)

const EmptyRepo = "QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn"

func main() {
	remoteName := os.Args[2]
	if strings.HasPrefix(remoteName, "pfg://") {
		remoteName = remoteName[len("pfg://"):]
	}

	if remoteName == "" {
		remoteName = EmptyRepo
	}

	if os.Getenv("GIT_DIR") == "" {
		log.Fatal().Msg("missing repository path ($GIT_DIR)")
	}

	handler, err := gitremote.NewIpfsProtocol(remoteName)
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