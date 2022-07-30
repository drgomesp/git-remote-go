package gitremote

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type ProtocolHandler interface {
	Capabilities() []string

	List(forPush bool) ([]string, error)
}

type Protocol struct {
	prefix string

	handler ProtocolHandler
}

func NewProtocol(prefix string, handler ProtocolHandler) *Protocol {
	return &Protocol{
		prefix:  prefix,
		handler: handler,
	}
}

func (p *Protocol) Run(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
loop:
	for {
		command, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		command = strings.Trim(command, "\n")

		log.Info().Msgf("< %s", command)
		switch {
		case command == "capabilities":
			_, _ = io.WriteString(w, fmt.Sprintf("list\n"))
			_, _ = io.WriteString(w, fmt.Sprintf("push\n"))
			_, _ = io.WriteString(w, fmt.Sprintf("fetch\n"))
		case strings.HasPrefix(command, "list"):
			list, err := p.handler.List(strings.HasPrefix(command, "list for-push"))
			if err != nil {
				return err
			}
			for _, e := range list {
				_, _ = io.WriteString(w, fmt.Sprintf("%s\n", e))
			}
			log.Printf("\n")
		case command == "":
			fallthrough
		case command == "\n":
			log.Info().Msg("")
			break loop
		default:
			return fmt.Errorf("received unknown command %q", command)
		}
	}

	return nil
}
