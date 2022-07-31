package gitremotego

import (
	"github.com/go-git/go-git/v5"

	"github.com/ipfs-shipyard/git-remote-ipld/core"
)

type ProtocolHandler interface {
	Initialize(tracker *core.Tracker, repo *git.Repository) error
	Capabilities() []string
	List(forPush bool) ([]string, error)
	Push(localRef string, remoteRef string) (string, error)
	ProvideBlock(identifier string, tracker *core.Tracker) ([]byte, error)
	Finish() error
}
