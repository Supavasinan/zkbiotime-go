package zkbiotime

import (
	"bytes"
	"encoding/json"
)

// Paginated is the standard DRF list envelope returned by BioTime list endpoints.
type Paginated[T any] struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Msg      string  `json:"msg,omitempty"`
	Code     int     `json:"code,omitempty"`
	Data     []T     `json:"data"`
}

// Ref is a reference to another object. BioTime serializes FK fields two ways
// depending on the endpoint: as a bare id (detail/update responses, e.g.
// `"department": 1`) or as a nested object (list/create responses, e.g.
// `"department": {"id":1,"dept_code":"1","dept_name":"…"}`). Ref decodes both and
// always exposes ID; the nested payload, when present, is kept in Object.
type Ref struct {
	ID     int
	Object map[string]any
}

func (r *Ref) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '{' {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		r.Object = m
		if v, ok := m["id"].(float64); ok {
			r.ID = int(v)
		}
		return nil
	}
	return json.Unmarshal(b, &r.ID)
}

// RefList decodes a FK list that may arrive as `[1]`, `[{...}]`, a single `1`, or
// a single `{...}`. Used for an employee's `area`.
type RefList []Ref

func (l *RefList) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '[' {
		var raws []json.RawMessage
		if err := json.Unmarshal(b, &raws); err != nil {
			return err
		}
		out := make(RefList, 0, len(raws))
		for _, raw := range raws {
			var ref Ref
			if err := json.Unmarshal(raw, &ref); err != nil {
				return err
			}
			out = append(out, ref)
		}
		*l = out
		return nil
	}
	var ref Ref
	if err := json.Unmarshal(b, &ref); err != nil {
		return err
	}
	*l = RefList{ref}
	return nil
}

// ─── Read models ──────────────────────────────────────────────────────────────

type Employee struct {
	ID           int     `json:"id"`
	EmpCode      string  `json:"emp_code"`
	FirstName    string  `json:"first_name,omitempty"`
	LastName     string  `json:"last_name,omitempty"`
	Nickname     string  `json:"nickname,omitempty"`
	Department   Ref     `json:"department,omitempty"` // id or {id,dept_code,dept_name}
	DeptName     string  `json:"dept_name,omitempty"`
	Area         RefList `json:"area,omitempty"` // [id] or [{id,area_code,area_name}]
	AreaName     string  `json:"area_name,omitempty"`
	Position     Ref     `json:"position,omitempty"`
	PositionName string  `json:"position_name,omitempty"`
	Gender       string  `json:"gender,omitempty"`
	Birthday     string  `json:"birthday,omitempty"`
	HireDate     string  `json:"hire_date,omitempty"`
	Mobile       string  `json:"mobile,omitempty"`
	Email        string  `json:"email,omitempty"`
	CardNo       string  `json:"card_no,omitempty"`
	AppStatus    *int    `json:"app_status,omitempty"`
}

type Department struct {
	ID         int    `json:"id"`
	DeptCode   string `json:"dept_code"`
	DeptName   string `json:"dept_name"`
	ParentDept *Ref   `json:"parent_dept,omitempty"`
}

type Area struct {
	ID         int    `json:"id"`
	AreaCode   string `json:"area_code"`
	AreaName   string `json:"area_name"`
	ParentArea *Ref   `json:"parent_area,omitempty"`
}

type Position struct {
	ID             int    `json:"id"`
	PositionCode   string `json:"position_code"`
	PositionName   string `json:"position_name"`
	ParentPosition *Ref   `json:"parent_position,omitempty"`
}

type Resign struct {
	ID         int    `json:"id"`
	Employee   Ref    `json:"employee"` // id or nested employee object
	EmpCode    string `json:"emp_code,omitempty"`
	ResignDate string `json:"resign_date,omitempty"`
	ResignType *int   `json:"resign_type,omitempty"`
	Reason     string `json:"reason,omitempty"`
	DisableAtt *int   `json:"disableatt,omitempty"`
}

type Terminal struct {
	ID           int    `json:"id"`
	SN           string `json:"sn"`
	Alias        string `json:"alias,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	Area         Ref    `json:"area,omitempty"` // nested {id,area_code,area_name} on read
	AreaName     string `json:"area_name,omitempty"`
	State        *int   `json:"state,omitempty"`
	LastActivity string `json:"last_activity,omitempty"`
}

type Transaction struct {
	ID            int    `json:"id"`
	EmpCode       string `json:"emp_code"`
	PunchTime     string `json:"punch_time"`
	PunchState    string `json:"punch_state,omitempty"`
	TerminalSN    string `json:"terminal_sn,omitempty"`
	TerminalAlias string `json:"terminal_alias,omitempty"`
}

// ─── Write inputs ─────────────────────────────────────────────────────────────

// EmployeeCreate — only EmpCode, Department and Area are required. BioTime expects
// `area` as a list of area ids (e.g. []int{1}); `department` is a single id.
type EmployeeCreate struct {
	EmpCode    string `json:"emp_code"`
	Department int    `json:"department"`
	Area       []int  `json:"area"`
	FirstName  string `json:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty"`
	Nickname   string `json:"nickname,omitempty"`
	Position   *int   `json:"position,omitempty"`
	Gender     string `json:"gender,omitempty"`
	Birthday   string `json:"birthday,omitempty"`
	HireDate   string `json:"hire_date,omitempty"`
	Mobile     string `json:"mobile,omitempty"`
	Email      string `json:"email,omitempty"`
	CardNo     string `json:"card_no,omitempty"`
}

type DepartmentCreate struct {
	DeptCode   string `json:"dept_code"`
	DeptName   string `json:"dept_name"`
	ParentDept *int   `json:"parent_dept,omitempty"`
}

type AreaCreate struct {
	AreaCode   string `json:"area_code"`
	AreaName   string `json:"area_name"`
	ParentArea *int   `json:"parent_area,omitempty"`
}

type PositionCreate struct {
	PositionCode   string `json:"position_code"`
	PositionName   string `json:"position_name"`
	ParentPosition *int   `json:"parent_position,omitempty"`
}

type ResignCreate struct {
	Employee   int    `json:"employee"`
	ResignDate string `json:"resign_date,omitempty"`
	ResignType *int   `json:"resign_type,omitempty"`
	Reason     string `json:"reason,omitempty"`
	DisableAtt *int   `json:"disableatt,omitempty"`
}

type TerminalCreate struct {
	SN        string `json:"sn"`
	IPAddress string `json:"ip_address"`
	Alias     string `json:"alias"`
	Area      *int   `json:"area,omitempty"`
	Heartbeat *int   `json:"heartbeat,omitempty"`
}

// AdjustResignInput is the body for the employees bulk-resign action.
type AdjustResignInput struct {
	Employees  string `json:"employees"`
	ResignDate string `json:"resign_date"`
	ResignType int    `json:"resign_type"`
	Reason     string `json:"reason"`
	DisableAtt int    `json:"disableatt"`
}

// DelBioTemplateInput selects which biometric templates to delete.
type DelBioTemplateInput struct {
	Employees   string `json:"employees"`
	FingerPrint bool   `json:"finger_print,omitempty"`
	Face        bool   `json:"face,omitempty"`
	FingerVein  bool   `json:"finger_vein,omitempty"`
	Palm        bool   `json:"palm,omitempty"`
}
