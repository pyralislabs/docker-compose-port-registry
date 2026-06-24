package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

func TestTextRendererWithWarnings(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary:       model.Summary{Projects: 1, Bindings: 1},
		Warnings: []model.Warning{
			{Message: "plain warning"},
			{Message: "sourced warning", Source: &model.SourceRef{File: "/x.yaml", Line: 5}},
		},
	}
	if err := r.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "WARNING plain warning") {
		t.Errorf("expected plain warning in output: %s", output)
	}
	if !strings.Contains(output, "WARNING /x.yaml: sourced warning") {
		t.Errorf("expected sourced warning in output: %s", output)
	}
}

func TestTextRendererFixesSummary(t *testing.T) {
	var buf bytes.Buffer
	r := NewTextRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary: model.Summary{
			Projects:     1,
			Bindings:     1,
			Collisions:   1,
			FixesApplied: 2,
		},
		Collisions: []model.Collision{
			{ID: "c1", Protocol: "tcp",
				HostIP:    model.HostScopeInfo{Scope: model.HostIPv4Any, Canonical: "0.0.0.0"},
				Published: model.Interval{Start: 8080, End: 8080},
				Bindings: []model.Binding{
					{ProjectID: "p", Service: "s"},
				}},
		},
		Fixes: []model.Fix{
			{Binding: model.Binding{ProjectID: "p", Service: "s"}, Status: model.FixApplied},
			{Binding: model.Binding{ProjectID: "p", Service: "s"}, Status: model.FixApplied},
		},
	}
	if err := r.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Applied 2 fix(es)") {
		t.Errorf("expected 'Applied 2 fix(es)' in output: %s", output)
	}
}

func TestFormatHostIPAllScopes(t *testing.T) {
	tests := []struct {
		hostIP model.HostScopeInfo
		want   string
	}{
		{model.HostScopeInfo{Scope: model.HostAnyUnspecified}, "0.0.0.0"},
		{model.HostScopeInfo{Scope: model.HostIPv4Any}, "0.0.0.0"},
		{model.HostScopeInfo{Scope: model.HostIPv6Any, Canonical: "::"}, "[::]"},
		{model.HostScopeInfo{Scope: model.HostIPv6Specific, Canonical: "::1"}, "[::1]"},
		{model.HostScopeInfo{Scope: model.HostIPv4Specific, Canonical: "192.168.1.1"}, "192.168.1.1"},
		{model.HostScopeInfo{Scope: model.HostUnresolved, Address: "myhost"}, "myhost"},
		{model.HostScopeInfo{Scope: model.HostScope(99), Address: "fallback"}, "fallback"},
	}
	for _, tt := range tests {
		if got := formatHostIP(tt.hostIP); got != tt.want {
			t.Errorf("formatHostIP(%+v) = %q, want %q", tt.hostIP, got, tt.want)
		}
	}
}

func TestFormatIntervalShort(t *testing.T) {
	if got := formatIntervalShort(model.Interval{Start: 80, End: 80}); got != "80" {
		t.Errorf("expected 80, got %s", got)
	}
	if got := formatIntervalShort(model.Interval{Start: 8000, End: 8010}); got != "8000-8010" {
		t.Errorf("expected 8000-8010, got %s", got)
	}
}

func TestFormatSourceShort(t *testing.T) {
	if got := formatSourceShort(model.SourceRef{File: "/x.yaml"}); got != "/x.yaml" {
		t.Errorf("expected /x.yaml, got %s", got)
	}
	if got := formatSourceShort(model.SourceRef{File: "/x.yaml", Line: 42}); got != "/x.yaml:42" {
		t.Errorf("expected /x.yaml:42, got %s", got)
	}
}

func TestValidateJSON(t *testing.T) {
	good := []byte(`{"schema_version":"1","tool_version":"0.1.0","roots":[],"summary":{"projects":0,"bindings":0,"collisions":0,"warnings":0,"fixes_planned":0,"fixes_applied":0}}`)
	if err := ValidateJSON(good); err != nil {
		t.Errorf("expected valid JSON to parse, got %v", err)
	}

	bad := []byte(`not json`)
	if err := ValidateJSON(bad); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJSONRendererCompactsOutput(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf)
	report := &model.Report{
		SchemaVersion: "1",
		ToolVersion:   "0.1.0",
		Summary:       model.Summary{},
	}
	if err := r.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := parsed["schema_version"]; !ok {
		t.Error("expected schema_version field")
	}
}

func TestBuildReportWithMultipleProjects(t *testing.T) {
	projects := []model.Project{
		{ID: "p1", Bindings: []model.Binding{{}, {}}},
		{ID: "p2", Bindings: []model.Binding{{}}},
	}
	r := BuildReport(projects, nil, nil, nil, []string{"/root"})

	if r.Summary.Projects != 2 {
		t.Errorf("expected 2 projects, got %d", r.Summary.Projects)
	}
	if r.Summary.Bindings != 3 {
		t.Errorf("expected 3 bindings, got %d", r.Summary.Bindings)
	}
	if len(r.Roots) != 1 || r.Roots[0] != "/root" {
		t.Errorf("expected roots [/root], got %v", r.Roots)
	}
}
