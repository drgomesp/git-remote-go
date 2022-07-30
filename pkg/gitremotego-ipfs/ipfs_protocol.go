package gitremotego_ipfs

import (
	"fmt"
	"os"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rs/zerolog/log"

	"github.com/drgomesp/git-remote-go/pkg/gitremotego"
)

var _ gitremotego.ProtocolHandler = &IpfsProtocol{}

type IpfsProtocol struct {
	repo *git.Repository
}

func NewIpfsProtocol() (*IpfsProtocol, error) {
	cwd, _ := os.Getwd()

	localDir, err := gitremotego.GetLocalDir()
	if localDir == "" {
		localDir = cwd
	}

	repo, err := git.PlainOpen(localDir)
	if err == git.ErrWorktreeNotProvided {
		repoRoot, _ := path.Split(localDir)

		repo, err = git.PlainOpen(repoRoot)
		if err != nil {
			return nil, err
		}
	}

	return &IpfsProtocol{repo: repo}, nil
}

func (h *IpfsProtocol) Initialize(repo *git.Repository) error {
	h.repo = repo
	return nil
}

func (h *IpfsProtocol) Capabilities() []string {
	return gitremotego.DefaultCapabilities
}

func (h *IpfsProtocol) List(forPush bool) ([]string, error) {
	log.Info().Msgf("List(forPush=%v)", forPush)
	return nil, nil
}

func (h *IpfsProtocol) Push(local string, remote string) (string, error) {
	log.Info().Msgf("Push(localRef=%v, remoteRef=%v)", local, remote)

	localRef, err := h.repo.Reference(plumbing.ReferenceName(local), true)
	if err != nil {
		return "", fmt.Errorf("push: %v", err)
	}

	c, err := gitremotego.CidFromHex(localRef.Hash().String())

	return fmt.Sprintf("hash=%v cid=%v\n", localRef.Hash().String(), c.String()), nil
}

func (h *IpfsProtocol) Fetch(sha, ref string) error {
	log.Info().Msgf("Fetch(sha=%v, ref=%v)", sha, ref)
	return nil
}
