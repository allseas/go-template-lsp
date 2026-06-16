package utils

import "path/filepath"

// FilePathToURI converts an absolute filesystem path (as returned by
// token.FileSet.Position) to an LSP-compatible file:// URI.
//
// On Windows, filepath.ToSlash converts backslashes and the drive letter
// produces a path like "C:/foo/bar" which needs a leading slash to form a
// valid file:// URI ("file:///C:/foo/bar"). On Unix the path already starts
// with '/', so the extra slash is not added.
func FilePathToURI(path string) string {
	slashed := filepath.ToSlash(path)
	if len(slashed) > 0 && slashed[0] != '/' {
		slashed = "/" + slashed
	}
	return "file://" + slashed
}
