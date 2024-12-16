package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
)

type mockHttpClient struct {
	OnGet func(url string) (*http.Response, error)
}

func (m mockHttpClient) Get(url string) (*http.Response, error) {
	return m.OnGet(url)
}

func (m *mockDataGenerator) OnGetSuccess(url string) (*http.Response, error) {
	bundle := data.SigstoreBundle(nil)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(bundle.String())),
	}, nil
}
