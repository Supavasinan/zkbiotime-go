package zkbiotime

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// doList fetches one page. (Methods can't be generic in Go, so the generic work
// lives in package-level helpers that the resource methods call with a concrete T.)
func doList[T any](ctx context.Context, c *Client, path string, query url.Values) (*Paginated[T], error) {
	var out Paginated[T]
	if err := c.Do(ctx, "GET", path, query, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doListAll walks every page and returns all items.
func doListAll[T any](ctx context.Context, c *Client, path string, query url.Values) ([]T, error) {
	q := url.Values{}
	for k, v := range query {
		q[k] = v
	}
	page := 1
	if p := q.Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			page = n
		}
	}
	if q.Get("page_size") == "" {
		q.Set("page_size", strconv.Itoa(c.pageSize))
	}
	var all []T
	for {
		q.Set("page", strconv.Itoa(page))
		res, err := doList[T](ctx, c, path, q)
		if err != nil {
			return nil, err
		}
		all = append(all, res.Data...)
		if res.Next == nil || len(res.Data) == 0 {
			break
		}
		page++
	}
	return all, nil
}

func detailPath(base string, id any) string { return fmt.Sprintf("%s%v/", base, id) }

// Resource is a generic CRUD endpoint. T is the read model, C the create input.
type Resource[T any, C any] struct {
	c    *Client
	path string
}

// List fetches one page (the raw DRF envelope).
func (r *Resource[T, C]) List(ctx context.Context, query url.Values) (*Paginated[T], error) {
	return doList[T](ctx, r.c, r.path, query)
}

// ListAll collects every page into a single slice.
func (r *Resource[T, C]) ListAll(ctx context.Context, query url.Values) ([]T, error) {
	return doListAll[T](ctx, r.c, r.path, query)
}

// Get fetches a single record by id.
func (r *Resource[T, C]) Get(ctx context.Context, id any) (*T, error) {
	var out T
	if err := r.c.Do(ctx, "GET", detailPath(r.path, id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Create POSTs a new record.
func (r *Resource[T, C]) Create(ctx context.Context, body C) (*T, error) {
	var out T
	if err := r.c.Do(ctx, "POST", r.path, nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update PATCHes a record (partial). body may be a struct or a map.
func (r *Resource[T, C]) Update(ctx context.Context, id, body any) (*T, error) {
	var out T
	if err := r.c.Do(ctx, "PATCH", detailPath(r.path, id), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Put replaces a record (PUT).
func (r *Resource[T, C]) Put(ctx context.Context, id, body any) (*T, error) {
	var out T
	if err := r.c.Do(ctx, "PUT", detailPath(r.path, id), nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a record (HTTP 204).
func (r *Resource[T, C]) Delete(ctx context.Context, id any) error {
	return r.c.Do(ctx, "DELETE", detailPath(r.path, id), nil, nil, nil)
}

// ReadDeleteResource is a list/get/delete-only endpoint (raw punch transactions
// are not creatable or updatable via REST).
type ReadDeleteResource[T any] struct {
	c    *Client
	path string
}

func (r *ReadDeleteResource[T]) List(ctx context.Context, query url.Values) (*Paginated[T], error) {
	return doList[T](ctx, r.c, r.path, query)
}
func (r *ReadDeleteResource[T]) ListAll(ctx context.Context, query url.Values) ([]T, error) {
	return doListAll[T](ctx, r.c, r.path, query)
}
func (r *ReadDeleteResource[T]) Get(ctx context.Context, id any) (*T, error) {
	var out T
	if err := r.c.Do(ctx, "GET", detailPath(r.path, id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (r *ReadDeleteResource[T]) Delete(ctx context.Context, id any) error {
	return r.c.Do(ctx, "DELETE", detailPath(r.path, id), nil, nil, nil)
}

// EmployeesService is the employees endpoint plus its bulk/device actions.
type EmployeesService struct {
	Resource[Employee, EmployeeCreate]
}

// AdjustArea reassigns employees (comma-separated ids) to one or more areas.
func (s *EmployeesService) AdjustArea(ctx context.Context, employees, areas string) error {
	return s.c.Do(ctx, "POST", s.path+"adjust_area/", nil, map[string]string{"employees": employees, "areas": areas}, nil)
}

// AdjustDepartment reassigns employees to a department.
func (s *EmployeesService) AdjustDepartment(ctx context.Context, employees string, department any) error {
	return s.c.Do(ctx, "POST", s.path+"adjust_department/", nil, map[string]any{"employees": employees, "department": department}, nil)
}

// AdjustPosition reassigns employees to a position.
func (s *EmployeesService) AdjustPosition(ctx context.Context, employees string, position any) error {
	return s.c.Do(ctx, "POST", s.path+"adjust_position/", nil, map[string]any{"employees": employees, "position": position}, nil)
}

// AdjustResign resigns employees in bulk.
func (s *EmployeesService) AdjustResign(ctx context.Context, in AdjustResignInput) error {
	return s.c.Do(ctx, "POST", s.path+"adjust_resign/", nil, in, nil)
}

// ResyncToDevice re-pushes employees to their device(s).
func (s *EmployeesService) ResyncToDevice(ctx context.Context, employees string) error {
	return s.c.Do(ctx, "POST", s.path+"resync_to_device/", nil, map[string]string{"employees": employees}, nil)
}

// DelBioTemplate deletes biometric templates (finger / face / vein / palm).
func (s *EmployeesService) DelBioTemplate(ctx context.Context, in DelBioTemplateInput) error {
	return s.c.Do(ctx, "POST", s.path+"del_bio_template/", nil, in, nil)
}

// TerminalsService is the terminals endpoint plus device actions.
type TerminalsService struct {
	Resource[Terminal, TerminalCreate]
}

func (s *TerminalsService) Reboot(ctx context.Context, terminals string) error {
	return s.c.Do(ctx, "POST", s.path+"reboot/", nil, map[string]string{"terminals": terminals}, nil)
}
func (s *TerminalsService) UploadAll(ctx context.Context, terminals string) error {
	return s.c.Do(ctx, "POST", s.path+"upload_all/", nil, map[string]string{"terminals": terminals}, nil)
}
func (s *TerminalsService) ClearCommand(ctx context.Context, terminals string) error {
	return s.c.Do(ctx, "POST", s.path+"clear_command/", nil, map[string]string{"terminals": terminals}, nil)
}
