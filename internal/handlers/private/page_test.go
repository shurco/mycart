package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shurco/litecart/internal/testutil"
)

func TestPages(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/pages", Pages)

	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
	}{
		{"default pagination", "", 1, 20},
		{"custom pagination", "?page=1&limit=5", 1, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/pages"+tt.query, "", cookie)
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}

			var res struct {
				Result struct {
					Page  int `json:"page"`
					Limit int `json:"limit"`
				} `json:"result"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&res)

			if res.Result.Page != tt.wantPage {
				t.Errorf("page = %d, want %d", res.Result.Page, tt.wantPage)
			}
			if res.Result.Limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", res.Result.Limit, tt.wantLimit)
			}
		})
	}
}

func TestGetPage(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/pages/:page_id", GetPage)

	tests := []struct {
		name       string
		pageID     string
		wantStatus []int
	}{
		{"existing page from fixtures", "ig9jpCixAgAu31f", []int{http.StatusOK}},
		{"non-existent page", "nonexistent12345", []int{http.StatusNotFound, http.StatusInternalServerError}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/pages/"+tt.pageID, "", cookie)
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func TestPagesCRUD(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/_/pages", AddPage)
	app.Get("/api/_/pages/:page_id", GetPage)
	app.Patch("/api/_/pages/:page_id", UpdatePage)
	app.Patch("/api/_/pages/:page_id/content", UpdatePageContent)
	app.Patch("/api/_/pages/:page_id/active", UpdatePageActive)
	app.Delete("/api/_/pages/:page_id", DeletePage)

	// Create
	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/pages",
		`{"name":"TestPage","slug":"testcrud","position":"footer"}`, cookie)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: status = %d", resp.StatusCode)
	}
	var created struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()

	pageID := created.Result.ID
	if pageID == "" {
		t.Fatal("create returned empty id")
	}

	steps := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"get", http.MethodGet, "/api/_/pages/" + pageID, "", http.StatusOK},
		{"update", http.MethodPatch, "/api/_/pages/" + pageID, `{"name":"Updated","slug":"upd","position":"footer"}`, http.StatusOK},
		{"update content", http.MethodPatch, "/api/_/pages/" + pageID + "/content", `{"content":"<h1>Hello</h1>"}`, http.StatusOK},
		{"toggle active", http.MethodPatch, "/api/_/pages/" + pageID + "/active", "", http.StatusOK},
		{"delete", http.MethodDelete, "/api/_/pages/" + pageID, "", http.StatusOK},
	}

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, s.method, s.path, s.body, cookie)
			testutil.AssertStatus(t, resp, s.wantStatus)
		})
	}
}
