// Copyright (c) Rivestack
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.rivestack.io", "rsk_test_key", "1.0.0")
	if c.BaseURL != "https://api.rivestack.io" {
		t.Errorf("expected BaseURL %q, got %q", "https://api.rivestack.io", c.BaseURL)
	}
	if c.APIKey != "rsk_test_key" {
		t.Errorf("expected APIKey %q, got %q", "rsk_test_key", c.APIKey)
	}
	if c.UserAgent != "terraform-provider-rivestack/1.0.0" {
		t.Errorf("expected UserAgent %q, got %q", "terraform-provider-rivestack/1.0.0", c.UserAgent)
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://api.rivestack.io/", "rsk_test", "1.0.0")
	if c.BaseURL != "https://api.rivestack.io" {
		t.Errorf("expected BaseURL without trailing slash, got %q", c.BaseURL)
	}
}

func TestDoRequest_SetsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer rsk_test_key" {
			t.Errorf("expected Authorization header %q, got %q", "Bearer rsk_test_key", got)
		}
		if got := r.Header.Get("User-Agent"); got != "terraform-provider-rivestack/1.0.0" {
			t.Errorf("expected User-Agent header %q, got %q", "terraform-provider-rivestack/1.0.0", got)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test_key", "1.0.0")
	var result map[string]string
	err := c.doRequest(context.Background(), "GET", "/test", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status ok, got %q", result["status"])
	}
}

func TestDoRequest_404ReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   true,
			"code":    404,
			"message": "cluster not found",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	err := c.doRequest(context.Background(), "GET", "/api/ha/999", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true for error: %v", err)
	}
}

func TestDoRequest_409ReturnsConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   true,
			"code":    409,
			"message": "cluster has an active job",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	err := c.doRequest(context.Background(), "POST", "/api/ha/1/configure", map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsConflict(err) {
		t.Errorf("expected IsConflict to be true for error: %v", err)
	}
}

func TestDoRequest_SendsJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["name"] != "test-cluster" {
			t.Errorf("expected name %q, got %q", "test-cluster", body["name"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]int{"id": 1})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	var result map[string]int
	err := c.doRequest(context.Background(), "POST", "/api/ha/provision", map[string]string{"name": "test-cluster"}, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["id"] != 1 {
		t.Errorf("expected id 1, got %d", result["id"])
	}
}

func TestIsGone(t *testing.T) {
	err := &APIError{StatusCode: http.StatusGone, Message: "cluster already deleted"}
	if !IsGone(err) {
		t.Error("expected IsGone to be true")
	}
	if IsNotFound(err) {
		t.Error("expected IsNotFound to be false for 410")
	}
}

func TestGetCluster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ha/42" {
			t.Errorf("expected path /api/ha/42, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Cluster{
			ID:       42,
			TenantID: "rs-abc123",
			Name:     "test-cluster",
			Status:   "active",
			Region:   "eu-central",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	cluster, err := c.GetCluster(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cluster.ID != 42 {
		t.Errorf("expected ID 42, got %d", cluster.ID)
	}
	if cluster.TenantID != "rs-abc123" {
		t.Errorf("expected TenantID %q, got %q", "rs-abc123", cluster.TenantID)
	}
	if cluster.Status != "active" {
		t.Errorf("expected Status %q, got %q", "active", cluster.Status)
	}
}

func TestProvisionCluster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/ha/provision" {
			t.Errorf("expected path /api/ha/provision, got %s", r.URL.Path)
		}

		var req ProvisionClusterRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Name != "my-cluster" {
			t.Errorf("expected name %q, got %q", "my-cluster", req.Name)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ProvisionClusterResponse{
			ID:       1,
			TenantID: "rs-abc123",
			Name:     "my-cluster",
			Status:   "provisioning",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	resp, err := c.ProvisionCluster(context.Background(), ProvisionClusterRequest{
		Name:   "my-cluster",
		Region: "eu-central",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %d", resp.ID)
	}
	if resp.Status != "provisioning" {
		t.Errorf("expected Status %q, got %q", "provisioning", resp.Status)
	}
}

func TestDeleteCluster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/ha/42" {
			t.Errorf("expected path /api/ha/42, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "cluster deletion initiated"})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	err := c.DeleteCluster(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigureCluster(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ha/1/configure" {
			t.Errorf("expected path /api/ha/1/configure, got %s", r.URL.Path)
		}

		var req ConfigureRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Users) != 1 || req.Users[0].Username != "testuser" {
			t.Errorf("expected user testuser, got %+v", req.Users)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(ConfigureResponse{
			Message: "configuration initiated",
			JobID:   100,
			Users: []ConfigUserResponse{
				{Username: "testuser", Password: "generated_pass"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	resp, err := c.ConfigureCluster(context.Background(), 1, ConfigureRequest{
		Users: []ConfigUserRequest{{Username: "testuser"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.JobID != 100 {
		t.Errorf("expected JobID 100, got %d", resp.JobID)
	}
	if len(resp.Users) != 1 || resp.Users[0].Password != "generated_pass" {
		t.Errorf("expected generated password, got %+v", resp.Users)
	}
}

func TestGetBackupConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ha/1/backup-config" {
			t.Errorf("expected path /api/ha/1/backup-config, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(BackupConfig{
			ID:            1,
			ClusterID:     1,
			Enabled:       true,
			Schedule:      "0 3 * * *",
			RetentionFull: 14,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	config, err := c.GetBackupConfig(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !config.Enabled {
		t.Error("expected backup to be enabled")
	}
	if config.Schedule != "0 3 * * *" {
		t.Errorf("expected schedule %q, got %q", "0 3 * * *", config.Schedule)
	}
}

func TestGetServerTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ha/server-types" {
			t.Errorf("expected path /api/ha/server-types, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ServerTypesResponse{
			ServerTypes: []ServerType{
				{Type: "starter", Name: "Starter", CPUs: 4, MemoryGB: 8},
			},
			Default: "starter",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	resp, err := c.GetServerTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ServerTypes) != 1 {
		t.Fatalf("expected 1 server type, got %d", len(resp.ServerTypes))
	}
	if resp.ServerTypes[0].Type != "starter" {
		t.Errorf("expected type %q, got %q", "starter", resp.ServerTypes[0].Type)
	}
}

func TestGetExtensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ha/extensions" {
			t.Errorf("expected path /api/ha/extensions, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(ExtensionsResponse{
			Extensions: []Extension{
				{Name: "vector", Description: "Vector similarity search", Category: "ai", Default: true},
				{Name: "postgis", Description: "Geographic objects", Category: "geo", Default: false},
			},
			TotalCount: 2,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "rsk_test", "1.0.0")
	resp, err := c.GetExtensions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Extensions) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(resp.Extensions))
	}
	if resp.Extensions[0].Name != "vector" {
		t.Errorf("expected extension %q, got %q", "vector", resp.Extensions[0].Name)
	}
}
