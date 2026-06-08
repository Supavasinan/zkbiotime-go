# zkbiotime-go

> Typed Go client for the **ZKBioTime / BioTime 8** REST API — personnel, iClock devices, and attendance reports.

- ✅ **Typed & generic** — `Paginated[T]`, generic `Resource[T, C]` (List/Get/Create/Update/Delete).
- 🔁 **Pagination built in** — `List` (one page) and `ListAll` (walks every page).
- 🔐 **HTTP Basic auth** — the scheme that passes BioTime's `IsNotOpenAPI` license gate on trial *and* full licenses.
- 🧯 **Typed errors** — `*APIError` carries the status, body, and an `IsLicenseGate` flag.
- 📦 **Zero dependencies** — standard library only.

```bash
go get github.com/Supavasinan/zkbiotime-go@latest
```

## Quick start

```go
package main

import (
	"context"
	"fmt"

	zkbiotime "github.com/Supavasinan/zkbiotime-go"
)

func main() {
	zk, err := zkbiotime.New(zkbiotime.Options{
		BaseURL:  "http://10.10.10.218",
		Username: "admin",
		Password: "••••••",
	})
	if err != nil {
		panic(err)
	}
	ctx := context.Background()

	// List one page (typed)
	page, err := zk.Employees.List(ctx, nil)
	fmt.Println(page.Count, err)

	// Create — only EmpCode, Department and Area are required
	emp, err := zk.Employees.Create(ctx, zkbiotime.EmployeeCreate{
		EmpCode: "1001", Department: 1, Area: 1, FirstName: "Somchai",
	})

	// Update (PATCH) then delete
	_, _ = zk.Employees.Update(ctx, emp.ID, map[string]any{"position": 2})
	_ = zk.Employees.Delete(ctx, emp.ID)

	// Every punch across all pages
	punches, _ := zk.Transactions.ListAll(ctx, nil)
	fmt.Println(len(punches))
}
```

## Resources

| Field | Endpoint | Operations |
|---|---|---|
| `zk.Employees` | `/personnel/api/employees/` | CRUD + `AdjustArea`, `AdjustDepartment`, `AdjustPosition`, `AdjustResign`, `ResyncToDevice`, `DelBioTemplate` |
| `zk.Departments` | `/personnel/api/departments/` | CRUD |
| `zk.Areas` | `/personnel/api/areas/` | CRUD |
| `zk.Positions` | `/personnel/api/positions/` | CRUD |
| `zk.Resigns` | `/personnel/api/resigns/` | CRUD |
| `zk.Terminals` | `/iclock/api/terminals/` | CRUD + `Reboot`, `UploadAll`, `ClearCommand` |
| `zk.Transactions` | `/iclock/api/transactions/` | List / Get / Delete (raw punches aren't writable) |
| `zk.Reports` | `/att/api/<report>/` | `Get(report, query)`, `Export(report, query)` (`KnownReports` lists names) |

Query params use `net/url.Values`:

```go
import "net/url"

q := url.Values{}
q.Set("search", "somchai")
q.Set("page_size", "50")
page, _ := zk.Employees.List(ctx, q)
```

## Errors

```go
var apiErr *zkbiotime.APIError
if errors.As(err, &apiErr) {
	fmt.Println(apiErr.StatusCode, apiErr.Body)
	if apiErr.IsLicenseGate {
		// 403 on /personnel/ or /iclock/: the IsNotOpenAPI gate.
		// Use Basic auth (default) or enable the `api` license mod.
	}
}
```

## Authentication & the license gate

BioTime 8's `iclock` and `personnel` viewsets enforce an `IsNotOpenAPI` permission: requests carrying a **JWT or DRF token** get `403` unless the license includes the `api` mod (trial licenses don't). **HTTP Basic auth leaves `request.auth = None` and passes the gate**, so this client uses Basic auth. `/att/` report endpoints have no gate.

## Generic escape hatch

```go
var out map[string]any
err := zk.Do(ctx, "GET", "/personnel/api/employees/", url.Values{"page_size": {"1"}}, nil, &out)
```

## License

MIT
