package mcp

import (
	"reflect"
	"testing"
)

func TestNormalizeServerIDsDeduplicatesAndPreservesOrder(t *testing.T) {
	actual := normalizeServerIDs([]string{
		" server-a ",
		"server-b",
		"server-a",
		"",
		"server-c",
	})

	expected := []string{"server-a", "server-b", "server-c"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected normalized server ids: %#v", actual)
	}
}

func TestMergeHeadersKeepsExistingValues(t *testing.T) {
	existing := map[string]string{
		"Authorization": "Bearer old-token",
	}

	headers, err := mergeHeaders(existing, []HeaderInput{
		{
			Key:          "Authorization",
			KeepExisting: true,
		},
		{
			Key:   "X-Trace-ID",
			Value: "trace-001",
		},
	})
	if err != nil {
		t.Fatalf("mergeHeaders returned error: %v", err)
	}

	expected := map[string]string{
		"Authorization": "Bearer old-token",
		"X-Trace-ID":    "trace-001",
	}
	if !reflect.DeepEqual(headers, expected) {
		t.Fatalf("unexpected merged headers: %#v", headers)
	}
}

func TestMergeHeadersRejectsMissingValueForNewHeader(t *testing.T) {
	_, err := mergeHeaders(nil, []HeaderInput{
		{Key: "Authorization"},
	})
	if err == nil {
		t.Fatal("expected mergeHeaders to reject empty new header value")
	}
}

func TestNormalizeTransportTypeDefaultsToSSE(t *testing.T) {
	if actual := normalizeTransportType(""); actual != "sse" {
		t.Fatalf("unexpected default transport type: %q", actual)
	}
}

func TestNormalizeTransportTypeAcceptsHTTP(t *testing.T) {
	if actual := normalizeTransportType(" HTTP "); actual != "http" {
		t.Fatalf("unexpected normalized transport type: %q", actual)
	}
}
