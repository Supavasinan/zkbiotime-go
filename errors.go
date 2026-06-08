package zkbiotime

import (
	"fmt"
	"strings"
)

// APIError is returned for any non-2xx response from BioTime. It carries the
// status code and raw body, and flags the most common failure — the
// IsNotOpenAPI license gate (403 on /personnel/ and /iclock/ when the license
// lacks the `api` mod). See the package docs / the REST reference.
type APIError struct {
	StatusCode    int
	Body          string
	Method        string
	Path          string
	IsLicenseGate bool
}

func (e *APIError) Error() string {
	hint := ""
	if e.IsLicenseGate {
		hint = " — likely the IsNotOpenAPI license gate; use Basic auth (default) or enable the `api` license mod"
	}
	return fmt.Sprintf("zkbiotime: %s %s -> HTTP %d: %s%s", e.Method, e.Path, e.StatusCode, e.Body, hint)
}

func newAPIError(status int, body, method, path string) *APIError {
	return &APIError{
		StatusCode:    status,
		Body:          body,
		Method:        method,
		Path:          path,
		IsLicenseGate: status == 403 && (strings.HasPrefix(path, "/personnel/") || strings.HasPrefix(path, "/iclock/")),
	}
}
