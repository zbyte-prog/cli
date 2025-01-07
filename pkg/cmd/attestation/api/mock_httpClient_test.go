package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
	"github.com/golang/snappy"
	"github.com/stretchr/testify/mock"
)

type mockHttpClient struct {
	mock.Mock
	mutex             sync.RWMutex
	currNumCalls      int
	alwaysFail        bool
	failAfterNumCalls int
	OnGet             func(url string) (*http.Response, error)
}

func (m *mockHttpClient) Get(url string) (*http.Response, error) {
	m.On("OnGet").Return()
	m.mutex.Lock()
	m.currNumCalls++
	m.mutex.Unlock()

	if m.alwaysFail || (m.failAfterNumCalls > 0 && m.currNumCalls > m.failAfterNumCalls) {
		m.MethodCalled("OnGet")
		return &http.Response{
			StatusCode: 500,
		}, fmt.Errorf("failed to fetch with %s", url)
	}

	m.MethodCalled("OnGet")

	var compressed []byte
	compressed = snappy.Encode(compressed, data.SigstoreBundleRaw)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(compressed)),
	}, nil
}

func FailHTTPClient() mockHttpClient {
	return mockHttpClient{
		alwaysFail: true,
	}
}

func SuccessHTTPClient() mockHttpClient {
	return mockHttpClient{}
}

func HTTPClientFailsAfterNumCalls(numCalls int) mockHttpClient {
	return mockHttpClient{
		failAfterNumCalls: numCalls,
	}
}
