package gitremotego

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ipfs-shipyard/git-remote-ipld/core"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type Protocol struct {
	prefix   string
	localDir string

	tracker  *core.Tracker
	handler  ProtocolHandler
	repo     *git.Repository
	lazyWork []func() (string, error)
}

func NewProtocol(prefix string, handler ProtocolHandler) (*Protocol, error) {
	log.Info().Msgf("GIT_DIR=%s", os.Getenv("GIT_DIR"))

	localDir, err := GetLocalDir()
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainOpen(localDir)
	if err == git.ErrWorktreeNotProvided {
		repoRoot, _ := path.Split(localDir)

		repo, err = git.PlainOpen(repoRoot)
		if err != nil {
			return nil, err
		}
	}

	tracker, err := core.NewTracker(localDir)
	if err != nil {
		return nil, fmt.Errorf("fetch: %v", err)
	}

	if err = handler.Initialize(tracker, repo); err != nil {
		return nil, err
	}

	return &Protocol{
		prefix:   prefix,
		handler:  handler,
		repo:     repo,
		localDir: localDir,
		tracker:  tracker,
	}, nil
}

func (p *Protocol) Run(r io.Reader, w io.Writer) (err error) {
	reader := bufio.NewReader(r)
loop:
	for {
		command, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		command = strings.Trim(command, "\n")

		log.Info().Msgf("💻 %s", command)
		switch {
		case command == "capabilities":
			p.Printf(w, "push\n")
			p.Printf(w, "fetch\n")
			p.Printf(w, "\n")
		case strings.HasPrefix(command, "list"):
			list, err := p.handler.List(strings.HasPrefix(command, "list for-push"))
			if err != nil {
				log.Err(err).Send()
				return err
			}
			for _, e := range list {
				p.Printf(w, "%s\n", e)
			}
			p.Printf(w, "\n")
		case strings.HasPrefix(command, "push "):
			refs := strings.Split(command[5:], ":")
			p.push(refs[0], refs[1], false)
		case strings.HasPrefix(command, "fetch "):
			parts := strings.Split(command, " ")
			p.fetch(parts[1], parts[2])
		case command == "":
			fallthrough
		case command == "\n":
			log.Info().Msg("Processing tasks")
			for _, task := range p.lazyWork {
				resp, err := task()
				if err != nil {
					return err
				}
				p.Printf(w, "%s", resp)
			}
			p.Printf(w, "\n")
			p.lazyWork = nil
			break loop
		default:
			return fmt.Errorf("received unknown command %q", command)
		}
	}

	return p.handler.Finish()
}

func (p *Protocol) push(src string, dst string, force bool) {
	p.lazyWork = append(p.lazyWork, func() (string, error) {
		done, err := p.handler.Push(src, dst)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("ok %s\n", done), nil
	})
}

func (p *Protocol) fetch(sha string, ref string) {
	p.lazyWork = append(p.lazyWork, func() (string, error) {
		fetch := core.NewFetch(p.localDir, p.tracker, p.handler.ProvideBlock)

		if sha == "0000000000000000000000000000000000000000" {
			return "", nil
		}

		if err := fetch.FetchHash(sha); err != nil {
			return "", fmt.Errorf("fetch: %v", err)
		}

		sha, err := hex.DecodeString(sha)
		if err != nil {
			return "", fmt.Errorf("fetch: %v", err)
		}

		p.tracker.Set(ref, sha)

		return "", nil
	})
}

func (p *Protocol) Printf(w io.Writer, format string, a ...interface{}) (n int, err error) {

	return fmt.Fprintf(w, format, a...)
}