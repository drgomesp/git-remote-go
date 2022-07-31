package gitremotego_ipfs

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/ipfs/go-cid"
	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/rs/zerolog/log"
	gitv4 "gopkg.in/src-d/go-git.v4"

	"github.com/ipfs-shipyard/git-remote-ipld/core"

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

var _ gitremotego.ProtocolHandler = &IpfsProtocol{}

type IpfsProtocol struct {
	ipfs *ipfs.Shell
	repo *git.Repository

	tracker                 *core.Tracker
	didPush                 bool
	largeObjs               map[string]string
	remoteName, currentHash string
	localDir                string
}

func (p *IpfsProtocol) Finish() error {
	//TODO: publish
	if p.didPush {
		if err := p.fillMissingLobjs(p.tracker); err != nil {
			return err
		}

		log.Info().Msgf("Pushed to IPFS as ipld://%s", p.currentHash)
	}

	return nil
}

func (p *IpfsProtocol) fillMissingLobjs(tracker *core.Tracker) error {
	if p.largeObjs == nil {
		if err := p.loadObjectMap(); err != nil {
			return err
		}
	}

	tracked, err := tracker.ListPrefixed(LobjTrackerPrefix)
	if err != nil {
		return err
	}

	for k, v := range tracked {
		if _, has := p.largeObjs[k]; has {
			continue
		}

		k = strings.TrimPrefix(k, LobjTrackerPrefix+"/")

		p.largeObjs[k] = v
		p.currentHash, err = p.ipfs.PatchLink(p.currentHash, "objects/"+k, v, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *IpfsProtocol) loadObjectMap() error {
	p.largeObjs = map[string]string{}

	links, err := p.ipfs.List(p.currentHash + "/" + LargeObjectDir)
	if err != nil {
		//TODO: Find a better way with coreapi
		if isNoLink(err) {
			return nil
		}
		return err
	}

	for _, link := range links {
		p.largeObjs[link.Name] = link.Hash
	}

	return nil
}

func (p *IpfsProtocol) ProvideBlock(identifier string, tracker *core.Tracker) ([]byte, error) {
	if p.largeObjs == nil {
		if err := p.loadObjectMap(); err != nil {
			return nil, err
		}
	}

	mappedCid, ok := p.largeObjs[identifier]
	if !ok {
		return nil, core.ErrNotProvided
	}

	if err := p.tracker.Set(LobjTrackerPrefix+"/"+identifier, []byte(mappedCid)); err != nil {
		return nil, err
	}

	r, err := p.ipfs.Cat(fmt.Sprintf("/ipfs/%s", mappedCid))
	if err != nil {
		return nil, errors.New("cat error")
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	realCid, err := p.ipfs.DagPut(data, "raw", "git")
	if err != nil {
		return nil, err
	}

	if realCid != identifier {
		return nil, fmt.Errorf("unexpected cid for provided block %s != %s", realCid, identifier)
	}

	return data, nil
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

func (p *IpfsProtocol) Initialize(tracker *core.Tracker, repo *git.Repository) error {
	p.repo = repo
	p.ipfs = ipfs.NewShell("localhost:5001")

	if p.ipfs == nil {
		return errors.New("failed to initialize protocol shell")
	}

	p.currentHash = p.remoteName
	localDir, err := gitremotego.GetLocalDir()
	if err != nil {
		return err
	}

	repo, err = git.PlainOpen(localDir)
	if err != nil {
		return err
	}

	p.localDir = localDir
	p.repo = repo
	p.tracker = tracker

	return nil
}

func (p *IpfsProtocol) Capabilities() []string {
	return gitremotego.DefaultCapabilities
}

func (p *IpfsProtocol) List(forPush bool) ([]string, error) {
	out := make([]string, 0)

	if !forPush {
		refs, err := p.paths(p.ipfs, p.remoteName, 0)
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
				dest, err := p.getRef(r)
				if err != nil {
					return nil, err
				}

				dest = strings.Replace(dest, "ref: ", "@", 1)
				out = append(out, fmt.Sprintf("%s %s", dest, r))
			}
		}
	} else {
		it, err := p.repo.Branches()
		if err != nil {
			return nil, err
		}

		err = it.ForEach(func(ref *plumbing.Reference) error {
			remoteRef := "0000000000000000000000000000000000000000"

			localRef, err := p.ipfs.ResolvePath(path.Join(p.currentHash, ref.Name().String()))
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

func (p *IpfsProtocol) Push(local string, remote string) (string, error) {
	p.didPush = true

	localRef, err := p.repo.Reference(plumbing.ReferenceName(local), true)
	if err != nil {
		return "", fmt.Errorf("command push: %v", err)
	}

	headHash := localRef.Hash().String()
	repo, err := gitv4.PlainOpen(p.localDir)
	if err == git.ErrWorktreeNotProvided {
		repoRoot, _ := path.Split(p.localDir)

		repo, err = gitv4.PlainOpen(repoRoot)
		if err != nil {
			return "", err
		}
	}

	push := core.NewPush(p.localDir, p.tracker, repo)
	push.NewNode = p.bigNodePatcher(p.tracker)

	err = push.PushHash(headHash)
	if err != nil {
		return "", fmt.Errorf("command push: %v", err)
	}

	hash := localRef.Hash()
	p.tracker.Set(remote, (&hash)[:])

	c, err := core.CidFromHex(headHash)
	if err != nil {
		return "", fmt.Errorf("push: %v", err)
	}

	//patch object
	p.currentHash, err = p.ipfs.PatchLink(p.currentHash, remote, c.String(), true)
	if err != nil {
		return "", fmt.Errorf("push: %v", err)
	}

	head, err := p.getRef("HEAD")
	if err != nil {
		return "", fmt.Errorf("push: %v", err)
	}
	if head == "" {
		headRef, err := p.ipfs.Add(strings.NewReader("refs/heads/master")) //TODO: Make this smarter?
		if err != nil {
			return "", fmt.Errorf("push: %v", err)
		}

		p.currentHash, err = p.ipfs.PatchLink(p.currentHash, "HEAD", headRef, true)
		if err != nil {
			return "", fmt.Errorf("push: %v", err)
		}
	}

	return local, nil
}

// bigNodePatcher returns a function which patches large object mapping into
// the resulting object
func (p *IpfsProtocol) bigNodePatcher(tracker *core.Tracker) func(cid.Cid, []byte) error {
	return func(hash cid.Cid, data []byte) error {
		if len(data) > (1 << 21) {
			c, err := p.ipfs.Add(bytes.NewReader(data))
			if err != nil {
				return err
			}

			if err := tracker.Set(LobjTrackerPrefix+"/"+hash.String(), []byte(c)); err != nil {
				return err
			}

			p.currentHash, err = p.ipfs.PatchLink(p.currentHash, "objects/"+hash.String(), c, true)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func (p *IpfsProtocol) getRef(name string) (string, error) {
	r, err := p.ipfs.Cat(path.Join(p.remoteName, name))
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

func (p *IpfsProtocol) paths(api *ipfs.Shell, pathStr string, level int) ([]refPath, error) {
	links, err := api.List(pathStr)
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

			sub, err := p.paths(api, path.Join(pathStr, link.Name), level+1)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
		case ipfs.TFile:
			out = append(out, refPath{path.Join(pathStr, link.Name), RefPathRef, link.Hash})
		case -1, 0: //unknown, assume git node
			out = append(out, refPath{path.Join(pathStr, link.Name), RefPathHead, link.Hash})
		default:
			return nil, fmt.Errorf("unexpected link type %d", link.Type)
		}
	}
	return out, nil
}

func isNoLink(err error) bool {
	return strings.Contains(err.Error(), "no link named") || strings.Contains(err.Error(), "no link by that name")
}
