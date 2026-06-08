// Package zkbiotime is a typed Go client for the ZKBioTime / BioTime 8 REST API
// (personnel, iClock devices and attendance reports).
//
// It authenticates with HTTP Basic auth — the scheme that passes BioTime's
// IsNotOpenAPI license gate on trial and full licenses alike.
//
//	zk, _ := zkbiotime.New(zkbiotime.Options{
//		BaseURL:  "http://10.10.10.218",
//		Username: "admin",
//		Password: "••••••",
//	})
//	page, _ := zk.Employees.List(ctx, nil)
//	emp, _ := zk.Employees.Create(ctx, zkbiotime.EmployeeCreate{EmpCode: "1001", Department: 1, Area: 1})
package zkbiotime

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Options configures a Client.
type Options struct {
	// BaseURL of the BioTime server, e.g. "http://10.10.10.218". Required.
	BaseURL string
	// Username and Password for HTTP Basic auth. Required.
	Username string
	Password string
	// HTTPClient is optional; defaults to a client with a 30s timeout.
	HTTPClient *http.Client
	// PageSize is the default page size for ListAll (default 100).
	PageSize int
}

// Client is a BioTime 8 REST client. Construct it with New.
type Client struct {
	baseURL    string
	authHeader string
	http       *http.Client
	pageSize   int

	Employees    *EmployeesService
	Departments  *Resource[Department, DepartmentCreate]
	Areas        *Resource[Area, AreaCreate]
	Positions    *Resource[Position, PositionCreate]
	Resigns      *Resource[Resign, ResignCreate]
	Terminals    *TerminalsService
	Transactions *ReadDeleteResource[Transaction]
	Reports      *ReportsService
}

// New creates a Client.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("zkbiotime: BaseURL is required")
	}
	if opts.Username == "" {
		return nil, fmt.Errorf("zkbiotime: Username is required")
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	ps := opts.PageSize
	if ps <= 0 {
		ps = 100
	}
	c := &Client{
		baseURL:    strings.TrimRight(opts.BaseURL, "/"),
		authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte(opts.Username+":"+opts.Password)),
		http:       hc,
		pageSize:   ps,
	}
	c.Employees = &EmployeesService{Resource[Employee, EmployeeCreate]{c: c, path: "/personnel/api/employees/"}}
	c.Departments = &Resource[Department, DepartmentCreate]{c: c, path: "/personnel/api/departments/"}
	c.Areas = &Resource[Area, AreaCreate]{c: c, path: "/personnel/api/areas/"}
	c.Positions = &Resource[Position, PositionCreate]{c: c, path: "/personnel/api/positions/"}
	c.Resigns = &Resource[Resign, ResignCreate]{c: c, path: "/personnel/api/resigns/"}
	c.Terminals = &TerminalsService{Resource[Terminal, TerminalCreate]{c: c, path: "/iclock/api/terminals/"}}
	c.Transactions = &ReadDeleteResource[Transaction]{c: c, path: "/iclock/api/transactions/"}
	c.Reports = &ReportsService{c: c}
	return c, nil
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// Raw performs a request and returns the HTTP status and the undecoded response
// body, treating a non-2xx status as a normal result (no error). It's an escape
// hatch for passthrough/proxy scenarios where you need the exact status and bytes.
// Only a transport-level failure returns a non-nil error.
func (c *Client) Raw(ctx context.Context, method, path string, query url.Values, body any) (int, []byte, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("zkbiotime: encode body: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, data, nil
}

// Do performs a request against the BioTime server. If body is non-nil it is
// JSON-encoded; if out is non-nil the JSON response is decoded into it. A non-2xx
// status returns an *APIError.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	status, data, err := c.Raw(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return newAPIError(status, string(data), method, path)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("zkbiotime: decode %s: %w", path, err)
		}
	}
	return nil
}
