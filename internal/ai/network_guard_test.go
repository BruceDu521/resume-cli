package ai

import (
	"errors"
	"net/http"
	"os"
	"testing"
)

type deniedNetwork struct{}

func (deniedNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("external network is disabled in AI unit tests")
}
func TestMain(m *testing.M) {
	http.DefaultTransport = deniedNetwork{}
	http.DefaultClient.Transport = deniedNetwork{}
	os.Exit(m.Run())
}
