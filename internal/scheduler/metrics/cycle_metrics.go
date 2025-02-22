package metrics

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/armadaproject/armada/internal/scheduler/scheduling"
)

var (
	poolLabels                  = []string{poolLabel}
	poolAndQueueLabels          = []string{poolLabel, queueLabel}
	queueAndPriorityClassLabels = []string{queueLabel, priorityClassLabel}
	poolQueueAndResourceLabels  = []string{poolLabel, queueLabel, resourceLabel}
)

type perCycleMetrics struct {
	consideredJobs         *prometheus.GaugeVec
	fairShare              *prometheus.GaugeVec
	adjustedFairShare      *prometheus.GaugeVec
	actualShare            *prometheus.GaugeVec
	fairnessError          *prometheus.GaugeVec
	demand                 *prometheus.GaugeVec
	cappedDemand           *prometheus.GaugeVec
	queueWeight            *prometheus.GaugeVec
	rawQueueWeight         *prometheus.GaugeVec
	gangsConsidered        *prometheus.GaugeVec
	gangsScheduled         *prometheus.GaugeVec
	firstGangQueuePosition *prometheus.GaugeVec
	lastGangQueuePosition  *prometheus.GaugeVec
	perQueueCycleTime      *prometheus.GaugeVec
	loopNumber             *prometheus.GaugeVec
	evictedJobs            *prometheus.GaugeVec
	evictedResources       *prometheus.GaugeVec
	spotPrice              *prometheus.GaugeVec
}

func newPerCycleMetrics() *perCycleMetrics {
	consideredJobs := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "considered_jobs",
			Help: "Number of jobs considered",
		},
		poolAndQueueLabels,
	)

	fairShare := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "fair_share",
			Help: "Fair share of each queue",
		},
		poolAndQueueLabels,
	)

	adjustedFairShare := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "adjusted_fair_share",
			Help: "Adjusted Fair share of each queue",
		},
		poolAndQueueLabels,
	)

	actualShare := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "actual_share",
			Help: "Actual Fair share of each queue",
		},
		poolAndQueueLabels,
	)

	demand := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "demand",
			Help: "Demand of each queue",
		},
		poolAndQueueLabels,
	)

	cappedDemand := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "capped_demand",
			Help: "Capped Demand of each queue and pool.  This differs from demand in that it limits demand by scheduling constraints",
		},
		poolAndQueueLabels,
	)

	queueWeight := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "queue_weight",
			Help: "Weight of the queue after multipliers have been applied",
		},
		poolAndQueueLabels,
	)

	rawQueueWeight := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "raw_queue_weight",
			Help: "Weight of the queue before multipliers have been applied",
		},
		poolAndQueueLabels,
	)

	fairnessError := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "fairness_error",
			Help: "Cumulative delta between adjusted fair share and actual share for all users who are below their fair share",
		},
		[]string{poolLabel},
	)

	gangsConsidered := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "gangs_considered",
			Help: "Number of gangs considered in this scheduling cycle",
		},
		poolAndQueueLabels,
	)

	gangsScheduled := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "gangs_scheduled",
			Help: "Number of gangs scheduled in this scheduling cycle",
		},
		poolAndQueueLabels,
	)

	firstGangQueuePosition := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "first_gang_queue_position",
			Help: "First position in the scheduling loop where a gang was considered",
		},
		poolAndQueueLabels,
	)

	lastGangQueuePosition := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "last_gang_queue_position",
			Help: "Last position in the scheduling loop where a gang was considered",
		},
		poolAndQueueLabels,
	)

	perQueueCycleTime := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "per_queue_schedule_cycle_times",
			Help: "Per queue cycle time when in a scheduling round.",
		},
		poolAndQueueLabels,
	)

	loopNumber := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "loop_number",
			Help: "Number of scheduling loops in this cycle",
		},
		poolLabels,
	)

	evictedJobs := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "evicted_jobs",
			Help: "Number of jobs evicted in this cycle",
		},
		poolAndQueueLabels,
	)

	evictedResources := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "evicted_resources",
			Help: "Resources evicted in this cycle",
		},
		poolQueueAndResourceLabels,
	)

	spotPrice := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "spot_price",
			Help: "spot price for given pool",
		},
		poolLabels,
	)

	return &perCycleMetrics{
		consideredJobs:         consideredJobs,
		fairShare:              fairShare,
		adjustedFairShare:      adjustedFairShare,
		actualShare:            actualShare,
		demand:                 demand,
		cappedDemand:           cappedDemand,
		queueWeight:            queueWeight,
		rawQueueWeight:         rawQueueWeight,
		fairnessError:          fairnessError,
		gangsConsidered:        gangsConsidered,
		gangsScheduled:         gangsScheduled,
		firstGangQueuePosition: firstGangQueuePosition,
		lastGangQueuePosition:  lastGangQueuePosition,
		perQueueCycleTime:      perQueueCycleTime,
		loopNumber:             loopNumber,
		evictedJobs:            evictedJobs,
		evictedResources:       evictedResources,
		spotPrice:              spotPrice,
	}
}

type cycleMetrics struct {
	leaderMetricsEnabled bool

	scheduledJobs           *prometheus.CounterVec
	premptedJobs            *prometheus.CounterVec
	scheduleCycleTime       prometheus.Histogram
	reconciliationCycleTime prometheus.Histogram
	latestCycleMetrics      atomic.Pointer[perCycleMetrics]
}

func newCycleMetrics() *cycleMetrics {
	scheduledJobs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "scheduled_jobs",
			Help: "Number of events scheduled",
		},
		queueAndPriorityClassLabels,
	)

	premptedJobs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "preempted_jobs",
			Help: "Number of jobs preempted",
		},
		queueAndPriorityClassLabels,
	)

	scheduleCycleTime := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    prefix + "schedule_cycle_times",
			Help:    "Cycle time when in a scheduling round.",
			Buckets: prometheus.ExponentialBuckets(10.0, 1.1, 110),
		},
	)

	reconciliationCycleTime := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    prefix + "reconciliation_cycle_times",
			Help:    "Cycle time when in a scheduling round.",
			Buckets: prometheus.ExponentialBuckets(10.0, 1.1, 110),
		},
	)

	cycleMetrics := &cycleMetrics{
		leaderMetricsEnabled:    true,
		scheduledJobs:           scheduledJobs,
		premptedJobs:            premptedJobs,
		scheduleCycleTime:       scheduleCycleTime,
		reconciliationCycleTime: reconciliationCycleTime,
		latestCycleMetrics:      atomic.Pointer[perCycleMetrics]{},
	}
	cycleMetrics.latestCycleMetrics.Store(newPerCycleMetrics())
	return cycleMetrics
}

func (m *cycleMetrics) enableLeaderMetrics() {
	m.leaderMetricsEnabled = true
}

func (m *cycleMetrics) disableLeaderMetrics() {
	m.resetLeaderMetrics()
	m.leaderMetricsEnabled = false
}

func (m *cycleMetrics) resetLeaderMetrics() {
	m.premptedJobs.Reset()
	m.scheduledJobs.Reset()
	m.latestCycleMetrics.Store(newPerCycleMetrics())
}

func (m *cycleMetrics) ReportScheduleCycleTime(cycleTime time.Duration) {
	m.scheduleCycleTime.Observe(float64(cycleTime.Milliseconds()))
}

func (m *cycleMetrics) ReportReconcileCycleTime(cycleTime time.Duration) {
	m.reconciliationCycleTime.Observe(float64(cycleTime.Milliseconds()))
}

func (m *cycleMetrics) ReportSchedulerResult(result scheduling.SchedulerResult) {
	// Metrics that depend on pool
	currentCycle := newPerCycleMetrics()
	for _, schedContext := range result.SchedulingContexts {
		pool := schedContext.Pool
		for queue, queueContext := range schedContext.QueueSchedulingContexts {
			jobsConsidered := float64(len(queueContext.UnsuccessfulJobSchedulingContexts) + len(queueContext.SuccessfulJobSchedulingContexts))
			actualShare := schedContext.FairnessCostProvider.UnweightedCostFromQueue(queueContext)
			demand := schedContext.FairnessCostProvider.UnweightedCostFromAllocation(queueContext.Demand)
			cappedDemand := schedContext.FairnessCostProvider.UnweightedCostFromAllocation(queueContext.CappedDemand)

			currentCycle.consideredJobs.WithLabelValues(pool, queue).Set(jobsConsidered)
			currentCycle.fairShare.WithLabelValues(pool, queue).Set(queueContext.FairShare)
			currentCycle.adjustedFairShare.WithLabelValues(pool, queue).Set(queueContext.AdjustedFairShare)
			currentCycle.actualShare.WithLabelValues(pool, queue).Set(actualShare)
			currentCycle.demand.WithLabelValues(pool, queue).Set(demand)
			currentCycle.cappedDemand.WithLabelValues(pool, queue).Set(cappedDemand)
			currentCycle.queueWeight.WithLabelValues(pool, queue).Set(queueContext.Weight)
			currentCycle.rawQueueWeight.WithLabelValues(pool, queue).Set(queueContext.RawWeight)
		}
		currentCycle.fairnessError.WithLabelValues(pool).Set(schedContext.FairnessError())
		currentCycle.spotPrice.WithLabelValues(pool).Set(schedContext.SpotPrice)
	}

	for _, jobCtx := range result.ScheduledJobs {
		m.scheduledJobs.WithLabelValues(jobCtx.Job.Queue(), jobCtx.PriorityClassName).Inc()
	}

	for _, jobCtx := range result.PreemptedJobs {
		m.premptedJobs.WithLabelValues(jobCtx.Job.Queue(), jobCtx.PriorityClassName).Inc()
	}

	for pool, schedulingStats := range result.PerPoolSchedulingStats {
		for queue, s := range schedulingStats.StatsPerQueue {
			currentCycle.gangsConsidered.WithLabelValues(pool, queue).Set(float64(s.GangsConsidered))
			currentCycle.gangsScheduled.WithLabelValues(pool, queue).Set(float64(s.GangsScheduled))
			currentCycle.firstGangQueuePosition.WithLabelValues(pool, queue).Set(float64(s.FirstGangConsideredQueuePosition))
			currentCycle.lastGangQueuePosition.WithLabelValues(pool, queue).Set(float64(s.LastGangScheduledQueuePosition))
			currentCycle.perQueueCycleTime.WithLabelValues(pool, queue).Set(float64(s.Time.Milliseconds()))
		}

		currentCycle.loopNumber.WithLabelValues(pool).Set(float64(schedulingStats.LoopNumber))

		for queue, s := range schedulingStats.EvictorResult.GetStatsPerQueue() {
			currentCycle.evictedJobs.WithLabelValues(pool, queue).Set(float64(s.EvictedJobCount))

			for _, r := range s.EvictedResources.GetResources() {
				currentCycle.evictedResources.WithLabelValues(pool, queue, r.Name).Set(float64(r.RawValue))
			}
		}
	}
	m.latestCycleMetrics.Store(currentCycle)
}

func (m *cycleMetrics) describe(ch chan<- *prometheus.Desc) {
	if m.leaderMetricsEnabled {
		m.scheduledJobs.Describe(ch)
		m.premptedJobs.Describe(ch)
		m.scheduleCycleTime.Describe(ch)

		currentCycle := m.latestCycleMetrics.Load()
		currentCycle.consideredJobs.Describe(ch)
		currentCycle.fairShare.Describe(ch)
		currentCycle.adjustedFairShare.Describe(ch)
		currentCycle.actualShare.Describe(ch)
		currentCycle.fairnessError.Describe(ch)
		currentCycle.demand.Describe(ch)
		currentCycle.cappedDemand.Describe(ch)
		currentCycle.queueWeight.Describe(ch)
		currentCycle.rawQueueWeight.Describe(ch)
		currentCycle.gangsConsidered.Describe(ch)
		currentCycle.gangsScheduled.Describe(ch)
		currentCycle.firstGangQueuePosition.Describe(ch)
		currentCycle.lastGangQueuePosition.Describe(ch)
		currentCycle.perQueueCycleTime.Describe(ch)
		currentCycle.loopNumber.Describe(ch)
		currentCycle.evictedJobs.Describe(ch)
		currentCycle.evictedResources.Describe(ch)
		currentCycle.spotPrice.Describe(ch)
	}

	m.reconciliationCycleTime.Describe(ch)
}

func (m *cycleMetrics) collect(ch chan<- prometheus.Metric) {
	if m.leaderMetricsEnabled {
		m.scheduledJobs.Collect(ch)
		m.premptedJobs.Collect(ch)
		m.scheduleCycleTime.Collect(ch)

		currentCycle := m.latestCycleMetrics.Load()
		currentCycle.consideredJobs.Collect(ch)
		currentCycle.fairShare.Collect(ch)
		currentCycle.adjustedFairShare.Collect(ch)
		currentCycle.actualShare.Collect(ch)
		currentCycle.fairnessError.Collect(ch)
		currentCycle.demand.Collect(ch)
		currentCycle.cappedDemand.Collect(ch)
		currentCycle.rawQueueWeight.Collect(ch)
		currentCycle.queueWeight.Collect(ch)
		currentCycle.gangsConsidered.Collect(ch)
		currentCycle.gangsScheduled.Collect(ch)
		currentCycle.firstGangQueuePosition.Collect(ch)
		currentCycle.lastGangQueuePosition.Collect(ch)
		currentCycle.perQueueCycleTime.Collect(ch)
		currentCycle.loopNumber.Collect(ch)
		currentCycle.evictedJobs.Collect(ch)
		currentCycle.evictedResources.Collect(ch)
		currentCycle.spotPrice.Collect(ch)
	}

	m.reconciliationCycleTime.Collect(ch)
}
