package fix

import (
	"testing"

	"github.com/pyralis-labs/compose-port-registry/internal/model"
	"gopkg.in/yaml.v3"
)

func TestEditBeforeByService(t *testing.T) {
	a := Edit{Binding: model.Binding{ProjectID: "p", Service: "z", Source: model.SourceRef{File: "x"}}}
	b := Edit{Binding: model.Binding{ProjectID: "p", Service: "a", Source: model.SourceRef{File: "x"}}}

	if !editBefore(b, a) {
		t.Error("expected b (service a) before a (service z)")
	}
	if editBefore(a, b) {
		t.Error("expected a > b")
	}
}

func TestFindAndReplaceInDocumentNode(t *testing.T) {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{
		{Kind: yaml.SequenceNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "8080:80"},
		}},
	}}

	modified, err := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Error("expected modification through DocumentNode -> SequenceNode -> ScalarNode")
	}
}

func TestFindAndReplaceInSequenceNodeWithNonScalar(t *testing.T) {
	doc := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Value: ""},
		},
	}

	modified, err := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected no modification in empty sequence")
	}
}

func TestFindAndReplaceInMappingNodeNonPorts(t *testing.T) {
	doc := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "image"},
			{Kind: yaml.ScalarNode, Value: "nginx"},
		},
	}

	modified, _ := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if modified {
		t.Error("expected no modification (no ports key)")
	}
}

func TestFindAndReplaceScalarInMapping(t *testing.T) {
	doc := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "ports"},
			{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "8080:80"},
			}},
		},
	}

	modified, _ := findAndReplaceInNode(doc, "8080:80", "4000:80")
	if !modified {
		t.Error("expected modification in nested ports")
	}
}

func TestWriteTemporaryReadOnlyDir(t *testing.T) {
	_, err := writeTemporary("/nonexistent/dir/file.yaml", []byte("content"))
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}
