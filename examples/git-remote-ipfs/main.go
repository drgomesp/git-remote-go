package main

import (
	"fmt"
	"os"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rs/zerolog/log"

	gitremote "github.com/drgomesp/git-remote-go/pkg"
)

var _ gitremote.ProtocolHandler = &IpfsHandler{}

func init() {
	wdir, err := os.Getwd()
	if err != nil {
		log.Err(err).Send()
	}

	if err = os.Setenv(
		"GIT_DIR",
		path.Join(wdir, "examples/git-remote-ipfs/git"),
	); err != nil {
		log.Err(err).Send()
	}
}

type IpfsHandler struct {
	repo *git.Repository
}

func (h *IpfsHandler) Initialize() error {
	localDir, err := GetLocalDir()
	if err != nil {
		return err
	}

	repo, err := git.PlainOpen(localDir)
	if err == git.ErrWorktreeNotProvided {
		repoRoot, _ := path.Split(localDir)

		repo, err = git.PlainOpen(repoRoot)
		if err != nil {
			return err
		}
	}

	h.repo = repo

	return nil
}

func (h *IpfsHandler) Capabilities() []string {
	return gitremote.DefaultCapabilities
}

func (h *IpfsHandler) List(forPush bool) ([]string, error) {
	log.Info().Msgf("List(forPush=%v)", forPush)
	return nil, nil
}

func (h *IpfsHandler) Push(local string, remote string) (string, error) {
	log.Info().Msgf("Push(localRef=%v, remoteRef=%v)", local, remote)

	localRef, err := h.repo.Reference(plumbing.ReferenceName(local), true)
	if err != nil {
		return "", fmt.Errorf("command push: %v", err)
	}

	c, err := CidFromHex(localRef.Hash().String())

	return fmt.Sprintf("hash=%v cid=%v\n", localRef.Hash().String(), c.String()), nil
}

func (h *IpfsHandler) Fetch(sha, ref string) error {
	log.Info().Msgf("Fetch(sha=%v, ref=%v)", sha, ref)
	return nil
}

func main() {
	if os.Getenv("GIT_DIR") == "" {
		log.Fatal().Msg("missing repository path ($GIT_DIR)")
	}

	proto, err := gitremote.NewProtocol("gitremotego", &IpfsHandler{})
	if err != nil {
		log.Err(err).Send()
	}

	if err := proto.Run(os.Stdin, os.Stdout); err != nil {
		log.Err(err).Send()
	}
}