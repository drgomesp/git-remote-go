package gitremote

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type handlerImpl struct {
}

func (h *handlerImpl) Capabilities() []string {
	return DefaultCapabilities
}

func (h *handlerImpl) List(forPush bool) ([]string, error) {
	return []string{"@refs/heads/master HEAD"}, nil
}

func Test_Protocol(t *testing.T) {
	proto := &Protocol{
		prefix:  "gitremotego",
		handler: &handlerImpl{},
	}

	tests := []struct {
		name string
		in   string
		out  []string
	}{
		{
			name: "capabilities",
			in:   "capabilities",
			out:  DefaultCapabilities,
		},
		{
			name: "list",
			in:   CmdList,
			out: []string{
				"@refs/heads/master HEAD",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.in + "\n")
			var writer bytes.Buffer
			if err := proto.Run(reader, &writer); err != nil {
				assert.Equal(t, io.EOF, err)
			}

			want := strings.Join(tt.out, "\n")
			got := strings.TrimSpace(writer.String())

			assert.Equal(t, want, got)
		})
	}
}
