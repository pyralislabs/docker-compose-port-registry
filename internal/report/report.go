package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
)

type TextRenderer struct {
	W io.Writer
}

func NewTextRenderer(w io.Writer) *TextRenderer {
	return &TextRenderer{W: w}
}

func (r *TextRenderer) Render(report *model.Report) error {
	if len(report.Collisions) == 0 && len(report.Warnings) == 0 && len(report.Fixes) == 0 {
		fmt.Fprintf(r.W, "No collisions found across %d project(s), %d binding(s).\n",
			report.Summary.Projects, report.Summary.Bindings)
		return nil
	}

	for _, col := range report.Collisions {
		hostIP := formatHostIP(col.HostIP)
		pubStr := formatIntervalShort(col.Published)
		fmt.Fprintf(r.W, "COLLISION %s %s:%s\n", col.Protocol, hostIP, pubStr)

		tw := tabwriter.NewWriter(r.W, 0, 0, 2, ' ', 0)
		for _, b := range col.Bindings {
			src := formatSourceShort(b.Source)
			pub := formatIntervalShort(b.Published)
			tgt := formatIntervalShort(b.Target)
			fmt.Fprintf(tw, "  %s/%s\t%s\t%s -> %s\n", b.ProjectID, b.Service, src, pub, tgt)
		}
		tw.Flush()
		fmt.Fprintln(r.W)
	}

	for _, w := range report.Warnings {
		if w.Source != nil {
			fmt.Fprintf(r.W, "WARNING %s: %s\n", w.Source.File, w.Message)
		} else {
			fmt.Fprintf(r.W, "WARNING %s\n", w.Message)
		}
	}

	if len(report.Fixes) > 0 {
		fmt.Fprintln(r.W, "")
		for _, f := range report.Fixes {
			switch f.Status {
			case model.FixPlanned:
				fmt.Fprintf(r.W, "PLANNED %s/%s: %s -> %s", f.Binding.ProjectID, f.Binding.Service, f.OldValue, f.NewValue)
			case model.FixApplied:
				fmt.Fprintf(r.W, "APPLIED %s/%s: %s -> %s", f.Binding.ProjectID, f.Binding.Service, f.OldValue, f.NewValue)
			case model.FixRefused:
				fmt.Fprintf(r.W, "REFUSED %s/%s: %s", f.Binding.ProjectID, f.Binding.Service, f.Reason)
			case model.FixRolledBack:
				fmt.Fprintf(r.W, "ROLLEDBACK %s/%s: %s", f.Binding.ProjectID, f.Binding.Service, f.Reason)
			}
			fmt.Fprintln(r.W)
		}
	}

	if len(report.Fixes) > 0 && report.Summary.FixesApplied > 0 {
		fmt.Fprintf(r.W, "\nApplied %d fix(es).\n", report.Summary.FixesApplied)
	}
	if report.Summary.Collisions > 0 && report.Summary.FixesApplied == 0 {
		fmt.Fprintf(r.W, "\nFound %d collision(s).\n", report.Summary.Collisions)
	}

	return nil
}

type JSONRenderer struct {
	W io.Writer
}

func NewJSONRenderer(w io.Writer) *JSONRenderer {
	return &JSONRenderer{W: w}
}

func (r *JSONRenderer) Render(report *model.Report) error {
	enc := json.NewEncoder(r.W)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func RenderReport(w io.Writer, report *model.Report, format string) error {
	if format == "json" {
		return NewJSONRenderer(w).Render(report)
	}
	return NewTextRenderer(w).Render(report)
}

func formatHostIP(h model.HostScopeInfo) string {
	switch h.Scope {
	case model.HostAnyUnspecified:
		return "0.0.0.0"
	case model.HostIPv4Any:
		return "0.0.0.0"
	case model.HostIPv6Any:
		return "[::]"
	case model.HostIPv4Specific:
		return h.Canonical
	case model.HostIPv6Specific:
		return "[" + h.Canonical + "]"
	case model.HostUnresolved:
		return h.Address
	default:
		return h.Address
	}
}

func formatIntervalShort(i model.Interval) string {
	if i.Start == i.End {
		return fmt.Sprintf("%d", i.Start)
	}
	return fmt.Sprintf("%d-%d", i.Start, i.End)
}

func formatSourceShort(s model.SourceRef) string {
	if s.Line > 0 {
		return fmt.Sprintf("%s:%d", s.File, s.Line)
	}
	return s.File
}

func BuildReport(projects []model.Project, collisions []model.Collision, warnings []model.Warning, fixes []model.Fix, roots []string) *model.Report {
	bindingCount := 0
	for _, p := range projects {
		bindingCount += len(p.Bindings)
	}

	report := &model.Report{
		SchemaVersion: model.SchemaVer,
		ToolVersion:   model.Version,
		Roots:         roots,
		Summary: model.Summary{
			Projects:     len(projects),
			Bindings:     bindingCount,
			Collisions:   len(collisions),
			Warnings:     len(warnings),
			FixesPlanned: model.CountFixesByStatus(fixes, model.FixPlanned),
			FixesApplied: model.CountFixesByStatus(fixes, model.FixApplied),
		},
		Collisions: collisions,
		Warnings:   warnings,
		Fixes:      fixes,
	}

	return report
}

func ValidateJSON(data []byte) error {
	var r model.Report
	return json.Unmarshal(data, &r)
}

func CollisionTextSummary(collisions []model.Collision) string {
	if len(collisions) == 0 {
		return "no collisions"
	}
	parts := make([]string, 0, len(collisions))
	for _, c := range collisions {
		parts = append(parts, string(c.ID))
	}
	return strings.Join(parts, ", ")
}
