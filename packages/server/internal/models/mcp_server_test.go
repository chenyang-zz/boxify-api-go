package models

import "testing"

func TestMCPMetasScansAndValuesJSON(t *testing.T) {
	var metas MCPMetas
	if err := metas.Scan([]byte(`[{"name":"search","description":"web search"}]`)); err != nil {
		t.Fatalf("Scan []byte error = %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "search" || metas[0].Description != "web search" {
		t.Fatalf("metas = %#v", metas)
	}

	value, err := metas.Value()
	if err != nil {
		t.Fatalf("Value error = %v", err)
	}
	if value != `[{"name":"search","description":"web search"}]` {
		t.Fatalf("value = %v", value)
	}
}

func TestMCPMetasScanHandlesStringAndNil(t *testing.T) {
	var metas MCPMetas
	if err := metas.Scan(`[{"name":"memory","description":"recall"}]`); err != nil {
		t.Fatalf("Scan string error = %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "memory" || metas[0].Description != "recall" {
		t.Fatalf("metas = %#v", metas)
	}

	if err := metas.Scan(nil); err != nil {
		t.Fatalf("Scan nil error = %v", err)
	}
	if len(metas) != 0 {
		t.Fatalf("nil scan metas = %#v, want empty", metas)
	}
}

func TestMCPMetaScanHandlesString(t *testing.T) {
	var meta MCPMeta
	if err := meta.Scan(`{"name":"search","description":"web search"}`); err != nil {
		t.Fatalf("Scan string error = %v", err)
	}
	if meta.Name != "search" || meta.Description != "web search" {
		t.Fatalf("meta = %#v", meta)
	}
}

func TestMCPAuthConfigScansAndValuesJSON(t *testing.T) {
	var config MCPAuthConfig
	if err := config.Scan([]byte(`{"token":"encrypted-token"}`)); err != nil {
		t.Fatalf("Scan []byte error = %v", err)
	}
	if config["token"] != "encrypted-token" {
		t.Fatalf("config = %#v", config)
	}

	value, err := config.Value()
	if err != nil {
		t.Fatalf("Value error = %v", err)
	}
	if value != `{"token":"encrypted-token"}` {
		t.Fatalf("value = %v", value)
	}
}

func TestMCPAuthConfigScanHandlesStringAndNil(t *testing.T) {
	var config MCPAuthConfig
	if err := config.Scan(`{"header":"X-Api-Key","key":"encrypted-key"}`); err != nil {
		t.Fatalf("Scan string error = %v", err)
	}
	if config["header"] != "X-Api-Key" || config["key"] != "encrypted-key" {
		t.Fatalf("config = %#v", config)
	}

	if err := config.Scan(nil); err != nil {
		t.Fatalf("Scan nil error = %v", err)
	}
	if len(config) != 0 {
		t.Fatalf("nil scan config = %#v, want empty", config)
	}
}
