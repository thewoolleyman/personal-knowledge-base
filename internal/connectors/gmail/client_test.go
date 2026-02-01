package gmail

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	gm "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func TestNewAPIClient_Success(t *testing.T) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewAPIClient_ServiceError(t *testing.T) {
	orig := createGmailService
	createGmailService = func(_ context.Context, _ ...option.ClientOption) (*gm.Service, error) {
		return nil, fmt.Errorf("service creation failed")
	}
	t.Cleanup(func() { createGmailService = orig })

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	_, err := NewAPIClient(context.Background(), ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create gmail service")
}

func TestSearchMessages_Success(t *testing.T) {
	// Mock the Gmail API: first a list call, then a get call for each message.
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if callCount == 0 {
			// List response
			fmt.Fprint(w, `{"messages":[{"id":"msg1","threadId":"t1"}]}`)
			callCount++
		} else {
			// Get response
			fmt.Fprint(w, `{"id":"msg1","snippet":"Test snippet","payload":{"headers":[{"name":"Subject","value":"Test Subject"},{"name":"From","value":"sender@example.com"}]}}`)
		}
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	messages, err := client.SearchMessages(context.Background(), "test")
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "msg1", messages[0].ID)
	assert.Equal(t, "Test Subject", messages[0].Subject)
	assert.Equal(t, "Test snippet", messages[0].Snippet)
	assert.Equal(t, "sender@example.com", messages[0].From)
}

func TestSearchMessages_ListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	_, err = client.SearchMessages(context.Background(), "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "gmail messages.list")
}

func TestSearchMessages_GetError_SkipsMessage(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if callCount == 0 {
			fmt.Fprint(w, `{"messages":[{"id":"msg1","threadId":"t1"}]}`)
			callCount++
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})
	client, err := NewAPIClient(context.Background(), ts)
	require.NoError(t, err)
	client.service.BasePath = srv.URL

	messages, err := client.SearchMessages(context.Background(), "test")
	require.NoError(t, err)
	assert.Empty(t, messages, "should skip messages that fail to fetch")
}
