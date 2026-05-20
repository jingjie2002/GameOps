package gameops

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	requests     atomic.Int64
	errors       atomic.Int64
	mails        atomic.Int64
	cdkRedeems   atomic.Int64
	gameEvents   atomic.Int64
	auditWrites  atomic.Int64
	riskAnalyses atomic.Int64
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP gameops_requests_total Total HTTP requests handled by GameOps.\n")
	fmt.Fprintf(w, "# TYPE gameops_requests_total counter\n")
	fmt.Fprintf(w, "gameops_requests_total %d\n", m.requests.Load())
	fmt.Fprintf(w, "# HELP gameops_errors_total Total failed HTTP requests.\n")
	fmt.Fprintf(w, "# TYPE gameops_errors_total counter\n")
	fmt.Fprintf(w, "gameops_errors_total %d\n", m.errors.Load())
	fmt.Fprintf(w, "# HELP gameops_mails_created_total Total reward mails created.\n")
	fmt.Fprintf(w, "# TYPE gameops_mails_created_total counter\n")
	fmt.Fprintf(w, "gameops_mails_created_total %d\n", m.mails.Load())
	fmt.Fprintf(w, "# HELP gameops_cdk_redeems_total Total successful CDK redemptions.\n")
	fmt.Fprintf(w, "# TYPE gameops_cdk_redeems_total counter\n")
	fmt.Fprintf(w, "gameops_cdk_redeems_total %d\n", m.cdkRedeems.Load())
	fmt.Fprintf(w, "# HELP gameops_events_total Total game events received.\n")
	fmt.Fprintf(w, "# TYPE gameops_events_total counter\n")
	fmt.Fprintf(w, "gameops_events_total %d\n", m.gameEvents.Load())
	fmt.Fprintf(w, "# HELP gameops_audit_writes_total Total audit log writes.\n")
	fmt.Fprintf(w, "# TYPE gameops_audit_writes_total counter\n")
	fmt.Fprintf(w, "gameops_audit_writes_total %d\n", m.auditWrites.Load())
	fmt.Fprintf(w, "# HELP gameops_risk_analyses_total Total AI risk analysis reports generated.\n")
	fmt.Fprintf(w, "# TYPE gameops_risk_analyses_total counter\n")
	fmt.Fprintf(w, "gameops_risk_analyses_total %d\n", m.riskAnalyses.Load())
}
