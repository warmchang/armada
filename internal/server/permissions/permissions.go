package permissions

import "github.com/armadaproject/armada/internal/common/auth/permission"

// Each principal (e.g., a user) has permissions associated with it.
// These are the possible permissions.
// For each gRPC call, the call handler first checks if the user has permissions for that call.
const (
	SubmitAnyJobs          permission.Permission = "submit_any_jobs"
	CancelAnyJobs                                = "cancel_any_jobs"
	PreemptAnyJobs                               = "preempt_any_jobs"
	ReprioritizeAnyJobs                          = "reprioritize_any_jobs"
	WatchAllEvents                               = "watch_all_events"
	CreateQueue                                  = "create_queue"
	DeleteQueue                                  = "delete_queue"
	CordonQueue                                  = "cordon_queue"
	CordonNodes                                  = "cordon_nodes"
	ExecuteJobs                                  = "execute_jobs"
	UpdateExecutorSettings                       = "update_executor_settings"
)
