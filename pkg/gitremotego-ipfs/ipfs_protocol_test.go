package gitremotego_ipfs_test

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	_ "github.com/joho/godotenv/autoload"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/drgomesp/git-remote-go/pkg/gitremotego"
	gitremotegoipfs "github.com/drgomesp/git-remote-go/pkg/gitremotego-ipfs"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func Test_IpfsProtocol(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
		err  error
	}{
		{
			name: "empty",
			in:   "",
			out:  ``,
		},
		{
			name: "capabilities",
			in:   "capabilities",
			out:  gitremotego.DefaultCapabilities,
		},
		{
			name: "push",
			in:   "push refs/heads/master:refs/heads/master\n",
			out:  `ok refs/heads/master`,
		},
		{
			name: "push fail",
			in:   "push a:b\n",
			out:  ``,
		},
		{
			name: "list",
			in:   "list\n",
			out: `ref: refs/heads/master HEAD
855496c865e07d73afd74a4de668658380ad6658 refs/heads/master`,
		},
		{
			name: "fetch",
			in:   "fetch f1932aa47ba178af518b36570dcc47c93575efb4 refs/heads/master\n",
			out:  ``,
		},
	}

	h, err := gitremotegoipfs.NewIpfsProtocol("QmUwjruL3Jy8vde2JiCAhvgh7TS8VgbTqApjooSqzm4bNm")
	proto, err := gitremotego.NewProtocol("gitremotego", h)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wdir, _ := os.Getwd()
			assert.NoError(t, os.RemoveAll(path.Join(wdir, "testrepo", "ipld")))

			assert.NoError(t, err)
			assert.NotNil(t, h)

			assert.NoError(t, err)

			reader := strings.NewReader(tt.in + "\n")
			var writer bytes.Buffer
			if err := proto.Run(reader, &writer); err != nil {
				if tt.err != io.EOF && tt.err != nil {
					assert.Equal(t, tt.err, err)
				}
			}

			got := strings.TrimSpace(writer.String())

			assert.Equal(t, tt.out, got)
		})
	}
}
