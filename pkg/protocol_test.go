package gitremote

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

type handlerMock struct {
	mock.Mock
}

func (h *handlerMock) Capabilities() []string {
	return DefaultCapabilities
}

func (h *handlerMock) List(forPush bool) ([]string, error) {
	args := h.Called(forPush)
	return args.Get(0).([]string), args.Error(1)
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