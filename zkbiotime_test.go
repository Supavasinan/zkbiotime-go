package zkbiotime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	c, err := New(Options{BaseURL: srv.URL, Username: "admin", Password: "p@ss"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, srv
}

func wantAuth(t *testing.T, r *http.Request) {
	t.Helper()
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:p@ss"))
	if got := r.Header.Get("Authorization"); got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestEmployeesList(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		wantAuth(t, r)
		if r.URL.Path != "/personnel/api/employees/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("page_size") != "5" {
			t.Errorf("page_size = %q", r.URL.Query().Get("page_size"))
		}
		// list-shape response: department is a nested object, area an array of objects
		_ = json.NewEncoder(w).Encode(map[string]any{
			"count": 1, "next": nil, "previous": nil,
			"data": []map[string]any{{
				"id": 1, "emp_code": "1001",
				"department": map[string]any{"id": 3, "dept_code": "1", "dept_name": "Dept"},
				"area":       []any{map[string]any{"id": 2, "area_code": "1", "area_name": "HQ"}},
			}},
		})
	})
	defer srv.Close()
	res, err := c.Employees.List(context.Background(), map[string][]string{"page_size": {"5"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 || len(res.Data) != 1 || res.Data[0].EmpCode != "1001" {
		t.Fatalf("bad result: %+v", res)
	}
	if res.Data[0].Department.ID != 3 {
		t.Errorf("Department.ID = %d, want 3", res.Data[0].Department.ID)
	}
	if len(res.Data[0].Area) != 1 || res.Data[0].Area[0].ID != 2 {
		t.Errorf("Area = %+v, want [{ID:2}]", res.Data[0].Area)
	}
}

func TestEmployeeCreate(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["emp_code"] != "1001" || body["department"] != float64(1) {
			t.Errorf("body = %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "emp_code": "1001"})
	})
	defer srv.Close()
	emp, err := c.Employees.Create(context.Background(), EmployeeCreate{EmpCode: "1001", Department: 1, Area: []int{1}})
	if err != nil {
		t.Fatal(err)
	}
	if emp.ID != 9 {
		t.Errorf("id = %d", emp.ID)
	}
}

func TestUpdatePatchAndDelete(t *testing.T) {
	var methods []string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.URL.Path != "/personnel/api/departments/5/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method == http.MethodPatch {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "dept_code": "D05", "dept_name": "X"})
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	})
	defer srv.Close()
	if _, err := c.Departments.Update(context.Background(), 5, map[string]any{"dept_name": "X"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Departments.Delete(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != http.MethodPatch || methods[1] != http.MethodDelete {
		t.Errorf("methods = %v", methods)
	}
}

func TestLicenseGateError(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail":"denied"}`))
	})
	defer srv.Close()
	_, err := c.Employees.List(context.Background(), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.StatusCode != 403 || !apiErr.IsLicenseGate {
		t.Errorf("license gate not flagged: %+v", apiErr)
	}
}

func TestListAllPaginates(t *testing.T) {
	n := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			next := "page2"
			_ = json.NewEncoder(w).Encode(Paginated[Transaction]{
				Count: 3, Next: &next,
				Data: []Transaction{{ID: 1, EmpCode: "1", PunchTime: "t"}, {ID: 2, EmpCode: "2", PunchTime: "t"}},
			})
		} else {
			_ = json.NewEncoder(w).Encode(Paginated[Transaction]{
				Count: 3, Next: nil,
				Data: []Transaction{{ID: 3, EmpCode: "3", PunchTime: "t"}},
			})
		}
	})
	defer srv.Close()
	all, err := c.Transactions.ListAll(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || n != 2 {
		t.Errorf("got %d items in %d requests", len(all), n)
	}
}

func TestReportsTargetAttApi(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/att/api/transactionReport/" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"count": 0, "next": nil, "previous": nil, "data": []any{}})
	})
	defer srv.Close()
	if _, err := c.Reports.Get(context.Background(), "transactionReport", nil); err != nil {
		t.Fatal(err)
	}
}

func TestRaw(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"nope"}`))
	})
	defer srv.Close()
	status, body, err := c.Raw(context.Background(), "GET", "/whatever/", nil, nil)
	if err != nil {
		t.Fatal(err) // Raw must NOT error on non-2xx
	}
	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
	if !strings.Contains(string(body), "nope") {
		t.Errorf("body = %s", body)
	}
}

// TestEmployeeFKShapes proves the polymorphic Ref decoding: BioTime returns FK
// fields as bare ids (detail/update) or nested objects (list/create).
func TestEmployeeFKShapes(t *testing.T) {
	// detail shape — bare ids
	var detail Employee
	if err := json.Unmarshal([]byte(`{"id":6,"emp_code":"e","department":1,"position":7,"area":[4]}`), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Department.ID != 1 || detail.Position.ID != 7 || len(detail.Area) != 1 || detail.Area[0].ID != 4 {
		t.Errorf("detail shape: dept=%d pos=%d area=%+v", detail.Department.ID, detail.Position.ID, detail.Area)
	}

	// list shape — nested objects
	var list Employee
	const body = `{"id":5,"emp_code":"e","department":{"id":1,"dept_code":"1","dept_name":"D"},"area":[{"id":2,"area_code":"1","area_name":"A"}]}`
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	if list.Department.ID != 1 || len(list.Area) != 1 || list.Area[0].ID != 2 {
		t.Errorf("list shape: dept=%d area=%+v", list.Department.ID, list.Area)
	}
	if list.Department.Object["dept_name"] != "D" {
		t.Errorf("nested object not preserved: %+v", list.Department.Object)
	}
}
