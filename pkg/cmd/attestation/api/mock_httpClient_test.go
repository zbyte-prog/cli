package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
	"github.com/golang/snappy"
)

type mockHttpClient struct {
	called bool
	OnGet  func(url string) (*http.Response, error)
}

func (m *mockHttpClient) Get(url string) (*http.Response, error) {
	m.called = true
	return m.OnGet(url)
}

func OnGetSuccess(url string) (*http.Response, error) {
	compressed := snappy.Encode(nil, data.SigstoreBundleRaw)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(compressed)),
	}, nil
}

func OnGetFail(url string) (*http.Response, error) {
	return &http.Response{
		StatusCode: 500,
	}, fmt.Errorf("failed to fetch with %s", url)
}
