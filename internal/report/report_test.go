package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestTextNoCollisions(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary: model.Summary{
			Projects: 2,
			Bindings: 4,
		},
	}
	err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "No collisions found") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestTextWithCollisions(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary: model.Summary{
			Projects:   2,
			Bindings:   2,
			Collisions: 1,
		},
		Collisions: []model.Collision{
			{
				ID:        "collision:tcp:ipv4-any:8080",
				Protocol:  "tcp",
				HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
				Published: model.Interval{Start: 8080, End: 8080},
				Bindings: []model.Binding{
					{ProjectID: "alpha", Service: "api", Published: model.Interval{Start: 8080, End: 8080}, Target: model.Interval{Start: 80, End: 80}},
					{ProjectID: "beta", Service: "web", Published: model.Interval{Start: 8080, End: 8080}, Target: model.Interval{Start: 3000, End: 3000}},
				},
			},
		},
	}
	err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "COLLISION") {
		t.Errorf("expected COLLISION in output: %s", output)
	}
	if !strings.Contains(output, "alpha/api") {
		t.Errorf("expected alpha/api in output: %s", output)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Roots:         []string{"/workspace"},
		Summary: model.Summary{
			Projects:   2,
			Bindings:   4,
			Collisions: 1,
		},
		Collisions: []model.Collision{
			{
				ID:        "collision:tcp:ipv4-any:8080",
				Protocol:  "tcp",
				HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
				Published: model.Interval{Start: 8080, End: 8080},
				Bindings: []model.Binding{
					{ProjectID: "alpha", Service: "api", Published: model.Interval{Start: 8080, End: 8080}, Target: model.Interval{Start: 80, End: 80}},
				},
			},
		},
	}
	err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded model.Report
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if decoded.SchemaVersion != "1" {
		t.Errorf("expected schema_version 1, got %s", decoded.SchemaVersion)
	}
	if len(decoded.Collisions) != 1 {
		t.Errorf("expected 1 collision, got %d", len(decoded.Collisions))
	}
}

func TestBuildReport(t *testing.T) {
	projects := []model.Project{
		{ID: "a", Bindings: []model.Binding{{ProjectID: "a"}}},
		{ID: "b", Bindings: []model.Binding{{ProjectID: "b"}}},
	}
	collisions := []model.Collision{
		{ID: "collision:tcp:ipv4-any:80", Protocol: "tcp", Published: model.Interval{Start: 80, End: 80}},
	}
	warnings := []model.Warning{
		{Message: "test warning"},
	}
	fixes := []model.Fix{
		{Binding: model.Binding{ProjectID: "a"}, OldValue: "80:80", NewValue: "4000:80", Status: model.FixPlanned},
	}

	r := BuildReport(projects, collisions, warnings, fixes, []string{"/test"})
	if r.Summary.Projects != 2 {
		t.Errorf("expected 2 projects, got %d", r.Summary.Projects)
	}
	if r.Summary.Collisions != 1 {
		t.Errorf("expected 1 collision, got %d", r.Summary.Collisions)
	}
	if r.Summary.Warnings != 1 {
		t.Errorf("expected 1 warning, got %d", r.Summary.Warnings)
	}
	if r.Summary.FixesPlanned != 1 {
		t.Errorf("expected 1 fix planned, got %d", r.Summary.FixesPlanned)
	}
}

func TestFormatHostIP(t *testing.T) {
	tests := []struct {
		hostIP model.HostScopeInfo
		want   string
	}{
		{model.HostScopeInfo{Scope: model.HostAnyUnspecified}, "0.0.0.0"},
		{model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"}, "0.0.0.0"},
		{model.HostScopeInfo{Scope: model.HostIPv6Any, Canonical: "::"}, "[::]"},
		{model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "192.168.1.1"}, "192.168.1.1"},
		{model.HostScopeInfo{Scope: model.HostUnresolved, Address: "hostname"}, "hostname"},
	}

	for _, tt := range tests {
		got := formatHostIP(tt.hostIP)
		if got != tt.want {
			t.Errorf("formatHostIP(%+v) = %q, want %q", tt.hostIP, got, tt.want)
		}
	}
}

func TestRenderReport(t *testing.T) {
	var buf bytes.Buffer
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary:       model.Summary{Projects: 1, Bindings: 1, Collisions: 0},
	}

	err := RenderReport(&buf, report, "json")
	if err != nil {
		t.Fatalf("JSON render error: %v", err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Error("invalid JSON output")
	}

	buf.Reset()
	err = RenderReport(&buf, report, "text")
	if err != nil {
		t.Fatalf("text render error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("empty text output")
	}
}

func TestCollisionTextSummary(t *testing.T) {
	summary := CollisionTextSummary(nil)
	if summary != "no collisions" {
		t.Errorf("expected 'no collisions', got %s", summary)
	}

	collisions := []model.Collision{
		{ID: "collision:tcp:ipv4-any:80"},
		{ID: "collision:tcp:ipv4-any:90"},
	}
	summary = CollisionTextSummary(collisions)
	if !strings.Contains(summary, "collision:tcp:ipv4-any:80") {
		t.Errorf("missing collision ID in summary: %s", summary)
	}
}

func TestWarningsAndFixes(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary: model.Summary{
			Projects:   1,
			Bindings:   1,
			Collisions: 1,
		},
		Collisions: []model.Collision{
			{
				ID:        "collision:tcp:ipv4-any:8080",
				Protocol:  "tcp",
				HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any},
				Published: model.Interval{Start: 8080, End: 8080},
				Bindings: []model.Binding{
					{ProjectID: "p", Service: "s", Published: model.Interval{Start: 8080, End: 8080}},
				},
			},
		},
		Fixes: []model.Fix{
			{Binding: model.Binding{ProjectID: "p", Service: "s"}, OldValue: "8080:80", NewValue: "4000:80", Status: model.FixApplied},
		},
	}
	err := r.Render(report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "APPLIED") {
		t.Errorf("expected APPLIED in output: %s", output)
	}
}
