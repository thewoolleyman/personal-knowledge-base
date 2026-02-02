package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cwoolley/personal-knowledge-base/internal/connectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_ReturnsResults(t *testing.T) {
	want := []connectors.Result{
		{Title: "Doc 1", Snippet: "snippet", URL: "https://example.com/1", Source: "gdrive"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "test query", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	got, err := c.Search(context.Background(), "test query", nil)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSearch_SendsSourcesParam(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gdrive", r.URL.Query().Get("sources"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]connectors.Result{})
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(context.Background(), "q", []string{"gdrive"})
	require.NoError(t, err)
}

func TestSearch_SendsMultipleSources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gdrive,gmail", r.URL.Query().Get("sources"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]connectors.Result{})
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(context.Background(), "q", []string{"gdrive", "gmail"})
	require.NoError(t, err)
}

func TestSearch_OmitsSourcesWhenNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, r.URL.Query().Has("sources"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]connectors.Result{})
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(context.Background(), "q", nil)
	require.NoError(t, err)
}

func TestSearch_ServerReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "all connectors failed"})
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(context.Background(), "q", nil)
	assert.ErrorContains(t, err, "all connectors failed")
}

func TestSearch_ServerReturnsBadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required parameter: q"})
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(context.Background(), "", nil)
	assert.ErrorContains(t, err, "missing required parameter: q")
}

func TestSearch_NetworkError(t *testing.T) {
	c := New("http://127.0.0.1:0", http.DefaultClient)
	_, err := c.Search(context.Background(), "q", nil)
	assert.Error(t, err)
}

func TestSearch_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(context.Background(), "q", nil)
	assert.Error(t, err)
}

func TestSearch_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := New(srv.URL, srv.Client())
	_, err := c.Search(ctx, "q", nil)
	assert.Error(t, err)
}
