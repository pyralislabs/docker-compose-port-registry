package model

func SortSourceRef(a, b SourceRef) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Column < b.Column
}

func HostScopeSortKey(h HostScopeInfo) string {
	return hostScopeSortKey(h)
}
