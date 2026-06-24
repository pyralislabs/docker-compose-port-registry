package fix

import (
	"strings"

	"gopkg.in/yaml.v3"
)

func yamlParse(s string) *yaml.Node {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(s), &doc); err != nil {
		panic(err)
	}
	return &doc
}

func yamlRender(n *yaml.Node) string {
	data, err := yaml.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
