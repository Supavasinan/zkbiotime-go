package zkbiotime

import (
	"context"
	"net/url"
)

// KnownReports lists the common /att/api/*Report/ endpoints. Any report name is
// accepted by ReportsService.Get; this slice is just for discovery.
var KnownReports = []string{
	"appReport", "dailyAbsentReport", "dailyActivityReport", "dailyEarlyOutReport",
	"dailyExceptionReport", "dailyLateInReport", "dailyLeaveReport", "dailyOvertimeReport",
	"departmentOvertimeReport", "departmentSummaryReport", "empScheduleReport", "empSummaryReport",
	"employeeLeaveReport", "employeeOvertimeReport", "firstInLastOutReport", "firstLastReport",
	"groupOvertimeReport", "groupSummaryReport", "monthlyAbsenceReport", "monthlyOvertimeReport",
	"monthlyPunchReport", "monthlyStatusReport", "monthlyWorkHoursReport", "punchParingReport",
	"scheduledPunchReport", "staffSummaryReport", "staffTransactionReport", "timeCardReport",
	"totalTimeCardReportV2", "transactionReport", "weeklyOvertimeReport", "weeklyWorkedHoursReport",
}

// ReportsService accesses the attendance report endpoints (/att/api/<report>/).
// These have no license gate (they work under any auth).
type ReportsService struct {
	c *Client
}

// ReportRow is one untyped report row.
type ReportRow = map[string]any

// Get fetches a report page. Report rows are untyped (they vary per report).
func (s *ReportsService) Get(ctx context.Context, report string, query url.Values) (*Paginated[ReportRow], error) {
	return doList[ReportRow](ctx, s.c, "/att/api/"+report+"/", query)
}

// Export triggers the report's server-side /export/.
func (s *ReportsService) Export(ctx context.Context, report string, query url.Values) error {
	return s.c.Do(ctx, "GET", "/att/api/"+report+"/export/", query, nil, nil)
}
