package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_ServesHTML(t *testing.T) {
	h := Handler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "<html")
	assert.Contains(t, html, "Search")
}

func TestHandler_HTMLContainsSearchForm(t *testing.T) {
	h := Handler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "<input", "should have a search input")
	assert.Contains(t, html, "google-drive", "should have google-drive source option")
	assert.Contains(t, html, "gmail", "should have gmail source option")
	assert.Contains(t, html, "obsidian", "should have obsidian source option")
	assert.Contains(t, html, "notion", "should have notion source option")
}

func TestHandler_GmailDefaultOff(t *testing.T) {
	h := Handler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Gmail checkbox should NOT have the checked attribute
	assert.Contains(t, html, `value="gmail"`)
	assert.NotContains(t, html, `value="gmail" checked`)

	// Drive checkbox SHOULD be checked by default
	assert.Contains(t, html, `value="google-drive" checked`)

	// Obsidian checkbox SHOULD be checked by default
	assert.Contains(t, html, `value="obsidian" checked`)

	// Notion checkbox should NOT have the checked attribute
	assert.Contains(t, html, `value="notion"`)
	assert.NotContains(t, html, `value="notion" checked`)
}

func TestHandler_HasCSS(t *testing.T) {
	h := Handler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "<style", "should have embedded CSS")
}

func TestHandler_HasJavaScript(t *testing.T) {
	h := Handler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "<script", "should have embedded JavaScript")
	assert.Contains(t, html, "/search?q=", "JS should call the search API")
}
