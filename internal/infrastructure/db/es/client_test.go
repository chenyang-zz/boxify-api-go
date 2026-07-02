package es_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	infraes "github.com/boxify/api-go/internal/infrastructure/db/es"
)

func TestClientPingAndRequests(t *testing.T) {
	// 验证 ES client 会按 Elasticsearch API 发送基础读写、索引和批量删除请求。
	var gotAuth string
	var gotIndexBody map[string]any
	var gotSearchBody map[string]any
	var gotCreateIndexBody map[string]any
	var gotDeleteByQueryBody map[string]any
	var gotUpdateByQueryBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = w.Write([]byte(`{"version":{"number":"8.0.0"}}`))
		case r.Method == http.MethodHead && r.URL.Path == "/docs":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/docs":
			if err := json.NewDecoder(r.Body).Decode(&gotCreateIndexBody); err != nil {
				t.Fatalf("decode create index body: %v", err)
			}
			_, _ = w.Write([]byte(`{"acknowledged":true}`))
		case r.Method == http.MethodPut && r.URL.Path == "/docs/_doc/1":
			if err := json.NewDecoder(r.Body).Decode(&gotIndexBody); err != nil {
				t.Fatalf("decode index body: %v", err)
			}
			_, _ = w.Write([]byte(`{"result":"created"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/docs/_doc/1":
			_, _ = w.Write([]byte(`{"_id":"1","found":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/docs/_search":
			if err := json.NewDecoder(r.Body).Decode(&gotSearchBody); err != nil {
				t.Fatalf("decode search body: %v", err)
			}
			_, _ = w.Write([]byte(`{"hits":{"total":{"value":1}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/docs/_delete_by_query":
			if err := json.NewDecoder(r.Body).Decode(&gotDeleteByQueryBody); err != nil {
				t.Fatalf("decode delete by query body: %v", err)
			}
			_, _ = w.Write([]byte(`{"deleted":2}`))
		case r.Method == http.MethodPost && r.URL.Path == "/docs/_update_by_query":
			if err := json.NewDecoder(r.Body).Decode(&gotUpdateByQueryBody); err != nil {
				t.Fatalf("decode update by query body: %v", err)
			}
			_, _ = w.Write([]byte(`{"updated":2}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/docs/_doc/1":
			_, _ = w.Write([]byte(`{"result":"deleted"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := infraes.NewClient(infraes.Config{
		URL:      server.URL,
		Username: "elastic",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("elastic:secret"))
	if gotAuth != wantAuth {
		t.Fatalf("authorization = %q, want %q", gotAuth, wantAuth)
	}

	exists, err := client.IndexExists(ctx, "docs")
	if err != nil {
		t.Fatalf("IndexExists error = %v", err)
	}
	if !exists {
		t.Fatal("IndexExists = false, want true")
	}

	createResp, err := client.CreateIndex(ctx, "docs", map[string]any{"mappings": map[string]any{"properties": map[string]any{"content": map[string]any{"type": "text"}}}})
	if err != nil {
		t.Fatalf("CreateIndex error = %v", err)
	}
	if createResp["acknowledged"] != true || gotCreateIndexBody["mappings"] == nil {
		t.Fatalf("create index resp/body = %#v %#v", createResp, gotCreateIndexBody)
	}

	indexResp, err := client.Index(ctx, "docs", "1", map[string]any{"title": "hello"})
	if err != nil {
		t.Fatalf("Index error = %v", err)
	}
	if indexResp["result"] != "created" || gotIndexBody["title"] != "hello" {
		t.Fatalf("index resp/body = %#v %#v", indexResp, gotIndexBody)
	}

	getResp, err := client.Get(ctx, "docs", "1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if getResp["_id"] != "1" || getResp["found"] != true {
		t.Fatalf("get resp = %#v", getResp)
	}

	searchResp, err := client.Search(ctx, "docs", map[string]any{"query": map[string]any{"match_all": map[string]any{}}})
	if err != nil {
		t.Fatalf("Search error = %v", err)
	}
	if _, ok := searchResp["hits"].(map[string]any); !ok || gotSearchBody["query"] == nil {
		t.Fatalf("search resp/body = %#v %#v", searchResp, gotSearchBody)
	}

	deleteByQueryResp, err := client.DeleteByQuery(ctx, "docs", map[string]any{"query": map[string]any{"term": map[string]any{"document_id": "doc-1"}}})
	if err != nil {
		t.Fatalf("DeleteByQuery error = %v", err)
	}
	if deleteByQueryResp["deleted"] != json.Number("2") || gotDeleteByQueryBody["query"] == nil {
		t.Fatalf("delete by query resp/body = %#v %#v", deleteByQueryResp, gotDeleteByQueryBody)
	}

	updateByQueryResp, err := client.UpdateByQuery(ctx, "docs", map[string]any{"script": map[string]any{"source": "ctx._source.kb_id = params.kb_id"}, "query": map[string]any{"term": map[string]any{"document_id": "doc-1"}}})
	if err != nil {
		t.Fatalf("UpdateByQuery error = %v", err)
	}
	if updateByQueryResp["updated"] != json.Number("2") || gotUpdateByQueryBody["script"] == nil || gotUpdateByQueryBody["query"] == nil {
		t.Fatalf("update by query resp/body = %#v %#v", updateByQueryResp, gotUpdateByQueryBody)
	}

	deleteResp, err := client.Delete(ctx, "docs", "1")
	if err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if deleteResp["result"] != "deleted" {
		t.Fatalf("delete resp = %#v", deleteResp)
	}
}

func TestClientIndexExistsReturnsFalseForMissingIndex(t *testing.T) {
	// 验证 IndexExists 会把 404 映射为 false，而不是普通错误。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		if r.Method != http.MethodHead || r.URL.Path != "/missing" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := infraes.NewClient(infraes.Config{URL: server.URL})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	exists, err := client.IndexExists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("IndexExists error = %v", err)
	}
	if exists {
		t.Fatal("IndexExists = true, want false")
	}
}

func TestNewClientSupportsAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		if got := r.Header.Get("Authorization"); got != "APIKey key-123" {
			t.Fatalf("authorization = %q", got)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, err := infraes.NewClient(infraes.Config{URL: server.URL, APIKey: "key-123"})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping error = %v", err)
	}
}

func TestNewClientRequiresURL(t *testing.T) {
	if _, err := infraes.NewClient(infraes.Config{}); err == nil || !strings.Contains(err.Error(), "Elasticsearch URL") {
		t.Fatalf("NewClient error = %v, want url error", err)
	}
}
