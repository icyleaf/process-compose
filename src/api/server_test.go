package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/docs"
	"github.com/gin-gonic/gin"
)

func TestStartHttpServerWithTCPUpdatesSwaggerHost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalHost := docs.SwaggerInfo.Host
	t.Cleanup(func() {
		docs.SwaggerInfo.Host = originalHost
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release test port: %v", err)
	}

	server, err := StartHttpServerWithTCP(false, "127.0.0.1", port, &mockProject{})
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	t.Cleanup(func() {
		_ = server.Close()
	})

	endpoint := fmt.Sprintf("http://127.0.0.1:%d/swagger/doc.json", port)

	var response *http.Response
	for i := 0; i < 20; i++ {
		response, err = http.Get(endpoint)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("failed to fetch swagger document: %v", err)
	}
	defer response.Body.Close()

	var document map[string]any
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatalf("failed to decode swagger document: %v", err)
	}

	expectedHost := fmt.Sprintf("127.0.0.1:%d", port)
	if got := document["host"]; got != expectedHost {
		t.Fatalf("expected swagger host %q, got %v", expectedHost, got)
	}
	if docs.SwaggerInfo.Host != expectedHost {
		t.Fatalf("expected swagger info host %q, got %q", expectedHost, docs.SwaggerInfo.Host)
	}
}
