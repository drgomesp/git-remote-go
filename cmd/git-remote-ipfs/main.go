package main

import (
	"os"

	"github.com/rs/zerolog/log"

	gitremote "github.com/drgomesp/git-remote-go/pkg"
)

type MyHandler struct {
}

func (m MyHandler) Capabilities() []string {
	panic("implement me")
}

func (m MyHandler) List(forPush bool) ([]string, error) {
	panic("implement me")
}

func main() {
	proto := gitremote.NewProtocol("gitremotego", &MyHandler{})

	if err := proto.Run(os.Stdin, os.Stdout); err != nil {
		log.Fatal().Err(err)
	}
}
