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
	Initialize() error
	Capabilities() []string
	List(forPush bool) ([]string, error)
	Push(localRef string, remoteRef string) (string, error)
	Fetch(sha, ref string) error
}

type Protocol struct {
	prefix string

	handler  ProtocolHandler
	lazyWork []func() (string, error)
}

func NewProtocol(prefix string, handler ProtocolHandler) (*Protocol, error) {
	if err := handler.Initialize(); err != nil {
		return nil, err
	}

	return &Protocol{
		prefix:  prefix,
		handler: handler,
	}, nil
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
			io.WriteString(w, strings.Join(p.handler.Capabilities(), "\n"))
		case strings.HasPrefix(command, CmdList):
			list, err := p.handler.List(strings.HasPrefix(command, "list for-push"))
			if err != nil {
				_, _ = io.WriteString(w, fmt.Sprintf("error: %s\n", err))
				return err
			}
			for _, e := range list {
				_, _ = io.WriteString(w, fmt.Sprintf("%s\n", e))
			}
			log.Printf("\n")
		case strings.HasPrefix(command, "push "):
			refs := strings.Split(command[5:], ":")
			p.push(refs[0], refs[1], false) //TODO: parse force
		case strings.HasPrefix(command, "fetch "):
			parts := strings.Split(command, " ")
			p.fetch(parts[1], parts[2])
		case command == "":
			fallthrough
		case command == "\n":
			log.Info().Msg("doing work...")

			for _, task := range p.lazyWork {
				resp, err := task()
				if err != nil {
					return err
				}

				io.WriteString(w, fmt.Sprintf("%s", resp))
			}

			p.lazyWork = nil
			break loop
		default:
			return fmt.Errorf("received unknown command %q", command)
		}
	}

	return nil
}

func (p *Protocol) push(src string, dst string, force bool) (string, error) {
	p.lazyWork = append(p.lazyWork, func() (string, error) {
		done, err := p.handler.Push(src, dst)
		if err != nil {
			log.Err(err).Msg("push failed")
			return "", err
		}

		return fmt.Sprintf("ok %s\n", done), nil
	})

	return "", nil
}

func (p *Protocol) fetch(s string, s2 string) error {
	return nil
}