package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// buildTestServer returns a tiny MCPServer with one tool so /rpc has something
// to call. We avoid pulling in the full mcptools package — the goal here is
// the HTTP shape, not the tool semantics.
func buildTestServer(t *testing.T) *server.MCPServer {
	t.Helper()
	srv := server.NewMCPServer("test", "0.0.0", server.WithToolCapabilities(true))
	tool := mcp.NewTool("echo", mcp.WithDescription("echo back text"),
		mcp.WithString("text", mcp.Required()))
	srv.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		text, _ := req.GetArguments()["text"].(string)
		return mcp.NewToolResultText(text), nil
	})
	return srv
}

func TestRPCRejectsGET(t *testing.T) {
	h := RPCHandler(buildTestServer(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/rpc", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /rpc -> %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got != "POST" {
		t.Fatalf("Allow header = %q, want POST", got)
	}
}

func TestRPCToolsList(t *testing.T) {
	h := RPCHandler(buildTestServer(t))
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/rpc", body)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type=%q, want application/json", got)
	}
	var resp struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("jsonrpc=%q, want 2.0", resp.JSONRPC)
	}
	if len(resp.Result.Tools) != 1 || resp.Result.Tools[0].Name != "echo" {
		t.Fatalf("expected one tool named 'echo', got %+v", resp.Result.Tools)
	}
}

func TestRPCToolCall(t *testing.T) {
	h := RPCHandler(buildTestServer(t))
	body := strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"text":"pong"}}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/rpc", body)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Result.IsError {
		t.Fatalf("isError=true, body=%s", rec.Body.String())
	}
	if len(resp.Result.Content) != 1 || resp.Result.Content[0].Text != "pong" {
		t.Fatalf("expected text content 'pong', got %+v", resp.Result.Content)
	}
}

func TestRPCBadJSONReturnsRPCError(t *testing.T) {
	// Bad JSON should NOT 4xx — by JSON-RPC convention, parse errors come
	// back as a JSON-RPC error object in a 200 response.
	h := RPCHandler(buildTestServer(t))
	body := strings.NewReader(`{not json`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/rpc", body)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200 (parse errors are JSON-RPC errors)", rec.Code)
	}
	var resp struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Error == nil {
		t.Fatalf("expected error object, got body=%s", rec.Body.String())
	}
	// JSON-RPC PARSE_ERROR code is -32700.
	if resp.Error.Code != -32700 {
		t.Fatalf("error code=%d, want -32700", resp.Error.Code)
	}
}

func TestRPCNotificationReturns204(t *testing.T) {
	// A JSON-RPC request without an "id" is a notification — no response body
	// by spec. We surface that as 204 No Content.
	h := RPCHandler(buildTestServer(t))
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/rpc", body)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func TestRPCBodyTooLargeReturns400(t *testing.T) {
	h := RPCHandler(buildTestServer(t))
	// Build a body well over the 4 MiB cap. The handler should refuse rather
	// than buffer arbitrary bytes from a misbehaving client.
	big := bytes.Repeat([]byte("a"), 5<<20)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/rpc", io.NopCloser(bytes.NewReader(big)))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 400 or 413", rec.Code)
	}
}
