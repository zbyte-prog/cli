package api

import (
	"bytes"
	"io"
	"net/http"

	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
	"github.com/golang/snappy"
)

type mockHttpClient struct {
	OnGet func(url string) (*http.Response, error)
}

func (m mockHttpClient) Get(url string) (*http.Response, error) {
	return m.OnGet(url)
}

func (m *mockDataGenerator) OnGetSuccess(url string) (*http.Response, error) {
	compressed := snappy.Encode(nil, data.SigstoreBundleRaw)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(compressed)),
	}, nil
}
