package scambus

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// TestServiceRoutes pins the method and path of every thin CRUD wrapper.
func TestServiceRoutes(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name   string
		method string
		path   string
		call   func(*Client) error
	}{
		{"journal delete", "DELETE", "/api/journal-entries/e1", func(c *Client) error {
			return c.Journal.Delete(ctx, "e1")
		}},
		{"journal in progress", "GET", "/api/journal-entries/in-progress", func(c *Client) error {
			_, err := c.Journal.InProgress(ctx)
			return err
		}},
		{"journal extracted identifiers", "GET", "/api/journal-entries/e1/extracted-identifiers", func(c *Client) error {
			_, err := c.Journal.ExtractedIdentifiers(ctx, "e1")
			return err
		}},
		{"external systems", "GET", "/api/external-systems", func(c *Client) error {
			_, err := c.Journal.ExternalSystems(ctx)
			return err
		}},

		{"identifier get", "GET", "/api/identifiers/i1", func(c *Client) error {
			_, err := c.Identifiers.Get(ctx, "i1")
			return err
		}},
		{"exclusion list", "GET", "/api/identifier-exclusions", func(c *Client) error {
			_, err := c.Identifiers.ListExclusions(ctx, 0, 0)
			return err
		}},
		{"exclusion delete", "DELETE", "/api/identifier-exclusions/x1", func(c *Client) error {
			return c.Identifiers.DeleteExclusion(ctx, "x1")
		}},

		{"case list", "GET", "/api/cases", func(c *Client) error {
			_, err := c.Cases.List(ctx, &ListCasesOptions{Status: "open"})
			return err
		}},
		{"case delete", "DELETE", "/api/cases/c1", func(c *Client) error {
			return c.Cases.Delete(ctx, "c1")
		}},
		{"case comments", "GET", "/api/cases/c1/comments", func(c *Client) error {
			_, err := c.Cases.ListComments(ctx, "c1")
			return err
		}},
		{"case comment create", "POST", "/api/cases/c1/comments", func(c *Client) error {
			_, err := c.Cases.CreateComment(ctx, "c1", "hi", "parent1")
			return err
		}},
		{"case comment count", "GET", "/api/cases/c1/comments/count", func(c *Client) error {
			_, err := c.Cases.CommentCount(ctx, "c1")
			return err
		}},
		{"comment update", "PUT", "/api/comments/cm1", func(c *Client) error {
			_, err := c.Comments.Update(ctx, "cm1", "edited")
			return err
		}},
		{"comment delete", "DELETE", "/api/comments/cm1", func(c *Client) error {
			return c.Comments.Delete(ctx, "cm1")
		}},

		{"queue list", "GET", "/api/queues", func(c *Client) error {
			_, err := c.Queues.List(ctx)
			return err
		}},
		{"queue create", "POST", "/api/queues", func(c *Client) error {
			_, err := c.Queues.Create(ctx, CreateQueueInput{Name: "q"})
			return err
		}},
		{"queue get", "GET", "/api/queues/q1", func(c *Client) error {
			_, err := c.Queues.Get(ctx, "q1")
			return err
		}},
		{"queue update", "PUT", "/api/queues/q1", func(c *Client) error {
			_, err := c.Queues.Update(ctx, "q1", QueuePatch{Name: Ptr("renamed")})
			return err
		}},
		{"queue delete", "DELETE", "/api/queues/q1", func(c *Client) error {
			return c.Queues.Delete(ctx, "q1")
		}},
		{"queue stats", "GET", "/api/queues/q1/stats", func(c *Client) error {
			_, err := c.Queues.Stats(ctx, "q1")
			return err
		}},
		{"queue items", "GET", "/api/queues/q1/items", func(c *Client) error {
			_, err := c.Queues.ListItems(ctx, "q1", "pending")
			return err
		}},
		{"queue release", "POST", "/api/queues/q1/items/i1/release", func(c *Client) error {
			return c.Queues.Release(ctx, "q1", "i1")
		}},
		{"queue contact", "POST", "/api/queues/q1/items/i1/contact", func(c *Client) error {
			return c.Queues.RecordContact(ctx, "q1", "i1", ContactInput{Notes: "called"})
		}},
		{"queue item history", "GET", "/api/queues/q1/items/i1/history", func(c *Client) error {
			_, err := c.Queues.ItemHistory(ctx, "q1", "i1")
			return err
		}},
		{"queue item events", "GET", "/api/queues/q1/items/i1/events", func(c *Client) error {
			_, err := c.Queues.ItemEvents(ctx, "q1", "i1")
			return err
		}},
		{"queue item cluster", "GET", "/api/queues/q1/items/i1/cluster", func(c *Client) error {
			_, err := c.Queues.ItemClusterIdentifiers(ctx, "q1", "i1", "")
			return err
		}},

		{"stream list", "GET", "/api/export-streams", func(c *Client) error {
			_, err := c.Streams.List(ctx, &ListStreamsOptions{ActiveOnly: true, Page: 1, PageSize: 5})
			return err
		}},
		{"stream get", "GET", "/api/export-streams/s1", func(c *Client) error {
			_, err := c.Streams.Get(ctx, "s1")
			return err
		}},
		{"stream delete", "DELETE", "/api/export-streams/s1", func(c *Client) error {
			return c.Streams.Delete(ctx, "s1")
		}},
		{"stream recover", "POST", "/api/export-streams/s1/recover", func(c *Client) error {
			_, err := c.Streams.Recover(ctx, "s1", true, false)
			return err
		}},
		{"stream recovery info", "GET", "/api/export-streams/s1/recovery-info", func(c *Client) error {
			_, err := c.Streams.RecoveryInfo(ctx, "s1")
			return err
		}},
		{"stream recovery history", "GET", "/api/redis/recovery/history", func(c *Client) error {
			_, err := c.Streams.RecoveryHistory(ctx, &RecoveryHistoryOptions{Limit: 10, Offset: 5, StreamID: "s1"})
			return err
		}},
		{"stream backfill", "POST", "/api/export-streams/s1/backfill-identifiers", func(c *Client) error {
			_, err := c.Streams.BackfillIdentifiers(ctx, "s1", "2025-01-01")
			return err
		}},
		{"consume info", "GET", "/api/consume/ck/info", func(c *Client) error {
			_, err := c.Consume.Info(ctx, "ck")
			return err
		}},

		{"file export create", "POST", "/api/file-exports", func(c *Client) error {
			_, err := c.FileExports.Create(ctx, CreateFileExportInput{SourceType: "view", EntityType: "identifier"})
			return err
		}},
		{"file export list", "GET", "/api/file-exports", func(c *Client) error {
			_, err := c.FileExports.List(ctx)
			return err
		}},
		{"file export get", "GET", "/api/file-exports/f1", func(c *Client) error {
			_, err := c.FileExports.Get(ctx, "f1")
			return err
		}},
		{"file export rename", "PATCH", "/api/file-exports/f1/rename", func(c *Client) error {
			_, err := c.FileExports.Rename(ctx, "f1", "new name")
			return err
		}},
		{"file export delete", "DELETE", "/api/file-exports/f1", func(c *Client) error {
			return c.FileExports.Delete(ctx, "f1")
		}},

		{"view list", "GET", "/api/views", func(c *Client) error {
			_, err := c.Views.List(ctx, nil)
			return err
		}},
		{"view get", "GET", "/api/views/v1", func(c *Client) error {
			_, err := c.Views.Get(ctx, "v1")
			return err
		}},
		{"view create", "POST", "/api/views", func(c *Client) error {
			_, err := c.Views.Create(ctx, CreateViewInput{Name: "v", EntityType: "identifier"})
			return err
		}},
		{"view update", "PUT", "/api/views/v1", func(c *Client) error {
			_, err := c.Views.Update(ctx, "v1", UpdateViewInput{Name: Ptr("renamed")})
			return err
		}},
		{"view my journal entries", "GET", "/api/views/my-journal-entries", func(c *Client) error {
			_, err := c.Views.MyJournalEntries(ctx)
			return err
		}},
		{"view my pinboard", "GET", "/api/views/my-pinboard", func(c *Client) error {
			_, err := c.Views.MyPinboard(ctx)
			return err
		}},

		{"tag list", "GET", "/api/tags", func(c *Client) error {
			_, err := c.Tags.List(ctx, nil)
			return err
		}},
		{"tag get", "GET", "/api/tags/t1", func(c *Client) error {
			_, err := c.Tags.Get(ctx, "t1")
			return err
		}},
		{"tag update", "PUT", "/api/tags/t1", func(c *Client) error {
			_, err := c.Tags.Update(ctx, "t1", UpdateTagInput{Active: Ptr(false)})
			return err
		}},
		{"tag delete", "DELETE", "/api/tags/t1", func(c *Client) error {
			return c.Tags.Delete(ctx, "t1")
		}},
		{"tag values list", "GET", "/api/tags/t1/values", func(c *Client) error {
			_, err := c.Tags.ListValues(ctx, "t1")
			return err
		}},
		{"tag value create", "POST", "/api/tags/t1/values", func(c *Client) error {
			_, err := c.Tags.CreateValue(ctx, "t1", CreateTagValue{Title: "Phishing"})
			return err
		}},
		{"tag value update", "PUT", "/api/tags/t1/values/v1", func(c *Client) error {
			_, err := c.Tags.UpdateValue(ctx, "t1", "v1", UpdateTagValueInput{Order: Ptr(2)})
			return err
		}},
		{"tag value delete", "DELETE", "/api/tags/t1/values/v1", func(c *Client) error {
			return c.Tags.DeleteValue(ctx, "t1", "v1")
		}},
		{"tag effective", "GET", "/api/tags/effective/journal_entry/e1", func(c *Client) error {
			_, err := c.Tags.Effective(ctx, "journal_entry", "e1")
			return err
		}},

		{"search cases", "POST", "/api/search/cases", func(c *Client) error {
			_, err := c.Search.Cases(ctx, SearchCasesInput{SearchQuery: "fraud"})
			return err
		}},

		{"notification list", "GET", "/api/notifications", func(c *Client) error {
			_, _, err := c.Notifications.List(ctx, &ListNotificationsOptions{UnreadOnly: true, Limit: 5, Offset: 2})
			return err
		}},
		{"notification get", "GET", "/api/notifications/n1", func(c *Client) error {
			_, err := c.Notifications.Get(ctx, "n1")
			return err
		}},
		{"notification mark read", "POST", "/api/notifications/n1/mark-read", func(c *Client) error {
			return c.Notifications.MarkRead(ctx, "n1")
		}},
		{"notification mark all read", "POST", "/api/notifications/mark-all-read", func(c *Client) error {
			return c.Notifications.MarkAllRead(ctx)
		}},
		{"notification dismiss", "POST", "/api/notifications/n1/dismiss", func(c *Client) error {
			return c.Notifications.Dismiss(ctx, "n1")
		}},
		{"notification dismiss all", "POST", "/api/notifications/dismiss-all", func(c *Client) error {
			return c.Notifications.DismissAll(ctx)
		}},

		{"session list", "GET", "/api/sessions", func(c *Client) error {
			_, err := c.Sessions.List(ctx)
			return err
		}},
		{"session revoke", "POST", "/api/sessions/s1/revoke", func(c *Client) error {
			return c.Sessions.Revoke(ctx, "s1")
		}},
		{"passkey list", "GET", "/api/passkeys", func(c *Client) error {
			_, err := c.Sessions.ListPasskeys(ctx)
			return err
		}},
		{"passkey delete", "DELETE", "/api/passkeys/p1", func(c *Client) error {
			return c.Sessions.DeletePasskey(ctx, "p1")
		}},
		{"2fa status", "GET", "/api/passkeys/2fa", func(c *Client) error {
			_, err := c.Sessions.TwoFactorStatus(ctx)
			return err
		}},
		{"2fa toggle", "POST", "/api/passkeys/2fa", func(c *Client) error {
			_, err := c.Sessions.SetTwoFactor(ctx, true)
			return err
		}},

		{"persona list", "GET", "/api/personas", func(c *Client) error {
			_, err := c.Personas.List(ctx)
			return err
		}},
		{"persona get", "GET", "/api/personas/p1", func(c *Client) error {
			_, err := c.Personas.Get(ctx, "p1")
			return err
		}},
		{"persona create", "POST", "/api/personas", func(c *Client) error {
			_, err := c.Personas.Create(ctx, CreatePersonaInput{Name: "Alice"})
			return err
		}},
		{"persona update", "PUT", "/api/personas/p1", func(c *Client) error {
			_, err := c.Personas.Update(ctx, "p1", UpdatePersonaInput{IsActive: Ptr(false)})
			return err
		}},
		{"persona delete", "DELETE", "/api/personas/p1", func(c *Client) error {
			return c.Personas.Delete(ctx, "p1")
		}},
		{"persona media update", "PUT", "/api/personas/p1/media/m1", func(c *Client) error {
			return c.Personas.UpdateMedia(ctx, "p1", "m1", UpdatePersonaMediaInput{Notes: Ptr("front")})
		}},
		{"persona media remove", "DELETE", "/api/personas/p1/media/m1", func(c *Client) error {
			return c.Personas.RemoveMedia(ctx, "p1", "m1")
		}},

		{"report journal entries", "POST", "/api/reports/journal-entries", func(c *Client) error {
			_, err := c.Reports.GenerateJournalEntryReport(ctx, JournalEntryReportInput{JournalEntryIDs: []string{"e1"}})
			return err
		}},
		{"report status", "GET", "/api/reports/r1/status", func(c *Client) error {
			_, err := c.Reports.Status(ctx, "r1")
			return err
		}},

		{"automation list", "GET", "/api/automations", func(c *Client) error {
			_, err := c.Automations.List(ctx, nil)
			return err
		}},
		{"automation get", "GET", "/api/automations/a1", func(c *Client) error {
			_, err := c.Automations.Get(ctx, "a1")
			return err
		}},
		{"automation create", "POST", "/api/automations", func(c *Client) error {
			_, err := c.Automations.Create(ctx, CreateAutomationInput{Name: "bot", IsActive: Ptr(true)})
			return err
		}},
		{"automation keys list", "GET", "/api/automations/a1/api-keys", func(c *Client) error {
			_, err := c.Automations.ListAPIKeys(ctx, "a1")
			return err
		}},
		{"automation key create", "POST", "/api/automations/a1/api-keys", func(c *Client) error {
			_, err := c.Automations.CreateAPIKey(ctx, "a1", "primary", "2026-01-01T00:00:00Z")
			return err
		}},
		{"automation key revoke", "POST", "/api/automations/a1/api-keys/k1/revoke", func(c *Client) error {
			return c.Automations.RevokeAPIKey(ctx, "a1", "k1")
		}},
		{"automation key delete", "DELETE", "/api/automations/a1/api-keys/k1", func(c *Client) error {
			return c.Automations.DeleteAPIKey(ctx, "a1", "k1")
		}},

		{"domain rule create", "POST", "/api/admin/special-domain-rules", func(c *Client) error {
			_, err := c.Admin.CreateSpecialDomainRule(ctx, CreateDomainRuleInput{Domain: "x.test", Category: "social"})
			return err
		}},
		{"domain rule update", "PUT", "/api/admin/special-domain-rules/r1", func(c *Client) error {
			_, err := c.Admin.UpdateSpecialDomainRule(ctx, "r1", UpdateDomainRuleInput{IsActive: Ptr(false)})
			return err
		}},
		{"domain rule delete", "DELETE", "/api/admin/special-domain-rules/r1", func(c *Client) error {
			return c.Admin.DeleteSpecialDomainRule(ctx, "r1")
		}},
		{"url consolidation start", "POST", "/api/admin/url-consolidation/start", func(c *Client) error {
			_, err := c.Admin.StartURLConsolidation(ctx)
			return err
		}},
		{"url consolidation status", "GET", "/api/admin/url-consolidation/status", func(c *Client) error {
			_, err := c.Admin.URLConsolidationStatus(ctx)
			return err
		}},
		{"url consolidation cancel", "POST", "/api/admin/url-consolidation/cancel", func(c *Client) error {
			return c.Admin.CancelURLConsolidation(ctx)
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, 200, stubBody(r.Method, r.URL.Path))
			})
			c := srv.client(t)

			if err := tc.call(c); err != nil {
				t.Fatalf("call: %v", err)
			}
			got := srv.last()
			if got.Method != tc.method || got.Path != tc.path {
				t.Fatalf("got %s %s, want %s %s", got.Method, got.Path, tc.method, tc.path)
			}
		})
	}
}

// stubBody answers with the shape the handler actually returns. Every entry
// here was read out of the backend handler, not assumed from the client.
func stubBody(method, path string) any {
	envelope := map[string]any{"data": []any{}, "pagination": map[string]any{}}

	switch path {
	// Paginated envelopes: tag.go:180, view.go:239, automation.go:86.
	case "/api/tags", "/api/views", "/api/automations":
		if method == http.MethodGet {
			return envelope
		}
	// notification.go:75.
	case "/api/notifications":
		return map[string]any{"notifications": []any{}, "total": 0, "limit": 0, "offset": 0}
	// session.go:49.
	case "/api/sessions":
		return map[string]any{"sessions": []any{}, "total": 0}
	// case_comment.go:236.
	case "/api/cases/c1/comments":
		if method == http.MethodGet {
			return map[string]any{"comments": []any{}, "total": 0}
		}
	// case_search.go:51.
	case "/api/search/cases":
		return map[string]any{"data": []any{}, "hasMore": false}
	// file_export.go:503 returns a presigned location, not bytes.
	case "/api/file-exports/f1/download":
		return map[string]any{"url": "https://storage.test/f1.csv", "file_name": "f1.csv"}
	}

	if method != http.MethodGet {
		return map[string]any{}
	}
	if bareArrayPaths[path] {
		return []any{}
	}
	return map[string]any{}
}

// bareArrayPaths are the endpoints that really do return a naked JSON array.
var bareArrayPaths = map[string]bool{
	"/api/journal-entries/in-progress":              true,
	"/api/journal-entries/e1/extracted-identifiers": true,
	"/api/external-systems":                         true,
	"/api/queues":                                   true,
	"/api/queues/q1/items":                          true,
	"/api/queues/q1/items/i1/history":               true,
	"/api/queues/q1/items/i1/events":                true,
	"/api/queues/q1/items/i1/cluster":               true,
	"/api/file-exports":                             true,
	"/api/tags/t1/values":                           true,
	"/api/tags/effective/journal_entry/e1":          true,
	"/api/passkeys":                                 true,
	"/api/personas":                                 true,
	"/api/automations/a1/api-keys":                  true,
	"/api/admin/special-domain-rules":               true,
}

func TestOptionsApplyToClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	httpClient := &http.Client{}
	c, err := New(
		WithAPIURL("https://example.test/api"),
		WithToken("t"),
		WithHTTPClient(httpClient),
		WithTimeout(7*time.Second),
		WithMaxRetries(3),
		WithRetryMaxTime(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient != httpClient {
		t.Fatal("WithHTTPClient did not take effect")
	}
	if c.timeout != 7*time.Second {
		t.Fatalf("timeout %v", c.timeout)
	}
	if httpClient.Timeout != 0 {
		t.Fatalf("WithTimeout must not mutate a caller-owned client, got %v", httpClient.Timeout)
	}
	if c.maxRetries != 3 || c.retryMaxTime != time.Minute {
		t.Fatalf("retry settings %d %v", c.maxRetries, c.retryMaxTime)
	}
}

func TestWithTimeoutIsOrderIndependent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	shared := &http.Client{Timeout: 90 * time.Second}

	for _, opts := range [][]Option{
		{WithHTTPClient(shared), WithTimeout(5 * time.Second)},
		{WithTimeout(5 * time.Second), WithHTTPClient(shared)},
	} {
		c, err := New(append([]Option{WithAPIURL("https://example.test/api"), WithToken("t")}, opts...)...)
		if err != nil {
			t.Fatal(err)
		}
		if c.timeout != 5*time.Second {
			t.Fatalf("timeout %v", c.timeout)
		}
		if shared.Timeout != 90*time.Second {
			t.Fatalf("caller client was mutated: %v", shared.Timeout)
		}
	}
}

func TestEnumHelpers(t *testing.T) {
	if !IdentifierTypePhone.Valid() || IdentifierType("nonsense").Valid() {
		t.Fatal("Valid is wrong")
	}
	if IdentifierTypeEmail.String() != "email" || EntryTypeDetection.String() != "detection" {
		t.Fatal("String is wrong")
	}
}
