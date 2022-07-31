package gitremotego_ipfs_test

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/drgomesp/git-remote-go/pkg/gitremotego"
	gitremotegoipfs "github.com/drgomesp/git-remote-go/pkg/gitremotego-ipfs"
)

func init() {
	getwd, err := os.Getwd()
	if err != nil {
		return
	}

	_ = os.Setenv("IPFS_PATH", "localhost:5001")
	_ = os.Setenv("GIT_DIR", path.Join(getwd, "testrepo"))

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func Test_IpfsProtocol(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []string
		err  error
	}{
		{
			name: "empty",
			in:   "",
			out:  nil,
		},
		{
			name: "capabilities",
			in:   "capabilities",
			out:  gitremotego.DefaultCapabilities,
		},
		{
			name: "push",
			in:   "push refs/heads/master:refs/heads/master\n",
			out:  []string{"ok refs/heads/master"},
		},
		{
			name: "push fail",
			in:   "push foo:bar\n",
			out:  []string{"push: reference not found"},
		},
		{
			name: "list",
			in:   "list\n",
			out: []string{
				"@refs/heads/master HEAD",
				"38f6d4ddae0b47b525b73aa9deaf36798fb30b7b refs/heads/master",
			},
		},
		{
			name: "fetch",
			in:   "fetch 38f6d4ddae0b47b525b73aa9deaf36798fb30b7b refs/heads/master\n",
			out: []string{
				"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := gitremotegoipfs.NewIpfsProtocol("QmWAz2sPeQy7nQNtMVo3NMZGDTsvX1cPVyTTPj3rB2VoVe")
			assert.NoError(t, err)
			assert.NotNil(t, h)

			proto, err := gitremotego.NewProtocol(
				"gitremotego",
				h,
			)
			assert.NoError(t, err)

			reader := strings.NewReader(tt.in + "\n")
			var writer bytes.Buffer
			if err := proto.Run(reader, &writer); err != nil {
				if tt.err != io.EOF && tt.err != nil {
					assert.Equal(t, tt.err, err)
				}
			}

			want := strings.Join(tt.out, "\n")
			got := strings.TrimSpace(writer.String())

			assert.Equal(t, want, got)
			wdir, err := os.Getwd()

			assert.NoError(t, os.RemoveAll(path.Join(wdir, "testrepo", "ipld")))
		})
	}
}
