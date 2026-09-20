package httpapi

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUnknownPublicationTimeIsNull(t *testing.T) {
	for _, value := range []any{Article{}, Reprint{}, StepReport{}} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if date, exists := fields["publishedAt"]; !exists || date != nil {
			t.Fatalf("%T: expected explicit null, got %s", value, encoded)
		}
	}
	at := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
	encoded, err := json.Marshal(Article{PublishedAt: at})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["publishedAt"] != at.Format(time.RFC3339) {
		t.Fatalf("known date lost: %s", encoded)
	}
}
