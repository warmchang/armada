package metrics

const (

	// common prefix for all metric names
	prefix = "armada_scheduler_"

	// Prometheus Labels
	checkpointLabel          = "checkpoint_interval"
	poolLabel                = "pool"
	typeLabel                = "type"
	priorityLabel            = "priority"
	queueLabel               = "queue"
	priorityClassLabel       = "priority_class"
	nodeLabel                = "node"
	nodeTypeLabel            = "nodeType"
	clusterLabel             = "cluster"
	errorCategoryLabel       = "category"
	errorSubcategoryLabel    = "subcategory"
	stateLabel               = "state"
	priorStateLabel          = "priorState"
	resourceLabel            = "resource"
	jobShapeLabel            = "job_shape"
	unschedulableReasonLabel = "unschedulable_reason"

	// Job state strings
	queued    = "queued"
	running   = "running"
	pending   = "pending"
	cancelled = "cancelled"
	leased    = "leased"
	preempted = "preempted"
	failed    = "failed"
	succeeded = "succeeded"

	noCheckpointLabelValue = "none"
)
