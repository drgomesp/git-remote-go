package gitremotego

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	getwd, err := os.Getwd()
	if err != nil {
		return
	}

	_ = os.Setenv("GIT_DIR", getwd)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

var _ ProtocolHandler = &handlerMock{}

type handlerMock struct {
	mock.Mock
}

func (h *handlerMock) Initialize(repo *git.Repository) error {
	return nil
}

func (h *handlerMock) Capabilities() []string {
	return DefaultCapabilities
}

func (h *handlerMock) List(forPush bool) ([]string, error) {
	args := h.Called(forPush)
	return args.Get(0).([]string), args.Error(1)
}

func (h *handlerMock) Push(localRef string, remoteRef string) (string, error) {
	args := h.Called(localRef, remoteRef)
	return args.String(0), args.Error(1)
}

func (h *handlerMock) Fetch(sha, ref string) error {
	args := h.Called(sha, ref)
	return args.Error(0)
}

func Test_Protocol(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  []string
		err  error
		mock func(m *handlerMock)
	}{
		{
			name: "empty",
			in:   "",
			out:  nil,
		},
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
			mock: func(m *handlerMock) {
				m.On("List", false).
					Return([]string{"@refs/heads/master HEAD"}, nil)
			},
		},
		{
			name: "list error",
			in:   CmdList,
			err:  errors.New("fail"),
			out:  []string{"error: fail"},
			mock: func(m *handlerMock) {
				m.
					On("List", false).
					Return([]string{}, errors.New("fail"))
			},
		},
		{
			name: "push",
			in:   "push a:b",
			out:  []string{"ok"},
			mock: func(m *handlerMock) {
				m.On("Push", "a", "b").
					Return([]string{"ok hash=26788196417edb6cc5d87d24a7c3be93ea79cf19 cid=baf4bcfbgpcazmql63nwmlwd5est4hput5j446gi"}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerMock := new(handlerMock)
			if tt.mock != nil {
				tt.mock(handlerMock)
			}

			proto := &Protocol{
				prefix:  "gitremotego",
				handler: handlerMock,
			}

			reader := strings.NewReader(tt.in + "\n")
			var writer bytes.Buffer
			if err := proto.Run(reader, &writer); err != nil {
				if tt.err != io.EOF && tt.err != nil {
					assert.Equal(t, tt.err, err)
				}
			}

			handlerMock.AssertExpectations(t)

			want := strings.Join(tt.out, "\n")
			got := strings.TrimSpace(writer.String())

			assert.Equal(t, want, got)
		})
	}
}
