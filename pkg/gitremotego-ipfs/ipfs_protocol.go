package gitremotego_ipfs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/rs/zerolog/log"

	"github.com/drgomesp/git-remote-go/pkg/gitremotego"
)

const (
	LargeObjectDir    = "objects"
	LobjTrackerPrefix = "//lobj"
)

const (
	RefPathHead = iota
	RefPathRef
)

type refPath struct {
	path  string
	rType int

	hash string
}

const RemotePeerforgePrefix = "pfg://"

var _ gitremotego.ProtocolHandler = &IpfsProtocol{}

type IpfsProtocol struct {
	ipfs *ipfs.Shell
	repo *git.Repository

	remoteName, currentHash string
}

func NewIpfsProtocol(remoteName string) (*IpfsProtocol, error) {
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

	return &IpfsProtocol{repo: repo, remoteName: remoteName}, nil
}

func (h *IpfsProtocol) Initialize(repo *git.Repository) error {
	h.repo = repo
	h.ipfs = ipfs.NewShell("localhost:5001")

	if h.ipfs == nil {
		return errors.New("failed to initialize protocol shell")
	}

	h.currentHash = h.remoteName
	localDir, err := gitremotego.GetLocalDir()
	if err != nil {
		return err
	}

	repo, err = git.PlainOpen(localDir)
	if err != nil {
		return err
	}

	h.repo = repo

	return err
	return nil
}

func (h *IpfsProtocol) Capabilities() []string {
	return gitremotego.DefaultCapabilities
}

func (h *IpfsProtocol) List(forPush bool) ([]string, error) {
	out := make([]string, 0)

	if !forPush {
		refs, err := h.paths(h.ipfs, h.remoteName, 0)
		if err != nil {
			return nil, err
		}

		for _, ref := range refs {
			switch ref.rType {
			case RefPathHead:
				r := path.Join(strings.Split(ref.path, "/")[1:]...)
				c, err := cid.Parse(ref.hash)
				if err != nil {
					return nil, err
				}

				hash, err := gitremotego.HexFromCid(c)
				if err != nil {
					return nil, err
				}

				out = append(out, fmt.Sprintf("%s %s", hash, r))
			case RefPathRef:
				r := path.Join(strings.Split(ref.path, "/")[1:]...)
				dest, err := h.getRef(r)
				if err != nil {
					return nil, err
				}
				out = append(out, fmt.Sprintf("@%s %s", dest, r))
			}

		}
	} else {
		it, err := h.repo.Branches()
		if err != nil {
			return nil, err
		}

		err = it.ForEach(func(ref *plumbing.Reference) error {
			remoteRef := "0000000000000000000000000000000000000000"

			localRef, err := h.ipfs.ResolvePath(path.Join(h.currentHash, ref.Name().String()))
			if err != nil && !isNoLink(err) {
				return err
			}
			if err == nil {
				refCid, err := cid.Parse(localRef)
				if err != nil {
					return err
				}

				remoteRef, err = gitremotego.HexFromCid(refCid)
				if err != nil {
					return err
				}
			}

			out = append(out, fmt.Sprintf("%s %s", remoteRef, ref.Name()))

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return out, nil
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

func (h *IpfsProtocol) getRef(name string) (string, error) {
	r, err := h.ipfs.Cat(path.Join(h.remoteName, name))
	if err != nil {
		if isNoLink(err) {
			return "", nil
		}
		return "", err
	}
	defer r.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (h *IpfsProtocol) paths(api *ipfs.Shell, p string, level int) ([]refPath, error) {
	links, err := api.List(p)
	if err != nil {
		return nil, err
	}

	out := make([]refPath, 0)
	for _, link := range links {
		switch link.Type {
		case ipfs.TDirectory:
			if level == 0 && link.Name == LargeObjectDir {
				continue
			}

			sub, err := h.paths(api, path.Join(p, link.Name), level+1)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
		case ipfs.TFile:
			out = append(out, refPath{path.Join(p, link.Name), RefPathRef, link.Hash})
		case -1, 0: //unknown, assume git node
			out = append(out, refPath{path.Join(p, link.Name), RefPathHead, link.Hash})
		default:
			return nil, fmt.Errorf("unexpected link type %d", link.Type)
		}
	}
	return out, nil
}

func isNoLink(err error) bool {
	return strings.Contains(err.Error(), "no link named") || strings.Contains(err.Error(), "no link by that name")
}
