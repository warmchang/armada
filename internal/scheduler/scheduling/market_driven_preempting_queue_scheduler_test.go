package scheduling

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/time/rate"
	v1 "k8s.io/api/core/v1"

	"github.com/armadaproject/armada/internal/common/armadacontext"
	armadamaps "github.com/armadaproject/armada/internal/common/maps"
	armadaslices "github.com/armadaproject/armada/internal/common/slices"
	"github.com/armadaproject/armada/internal/common/stringinterner"
	"github.com/armadaproject/armada/internal/common/types"
	"github.com/armadaproject/armada/internal/scheduler/configuration"
	"github.com/armadaproject/armada/internal/scheduler/internaltypes"
	"github.com/armadaproject/armada/internal/scheduler/jobdb"
	"github.com/armadaproject/armada/internal/scheduler/nodedb"
	schedulerconstraints "github.com/armadaproject/armada/internal/scheduler/scheduling/constraints"
	"github.com/armadaproject/armada/internal/scheduler/scheduling/context"
	"github.com/armadaproject/armada/internal/scheduler/scheduling/fairness"
	"github.com/armadaproject/armada/internal/scheduler/testfixtures"
	"github.com/armadaproject/armada/pkg/api"
	"github.com/armadaproject/armada/pkg/bidstore"
)

func TestMarketDrivenPreemptingQueueScheduler(t *testing.T) {
	type MarketData struct {
		// For each queue, expected billable resource that should be set on the queue context
		ExpectedBillableResource map[string]internaltypes.ResourceList
		// The expected spot price that should be set on the scheduling context
		ExpectedSpotPrice float64
	}
	type SchedulingRound struct {
		// Map from queue name to pod requirements for that queue.
		JobsByQueue map[string][]*jobdb.Job
		// For each queue, indices of jobs expected to be scheduled.
		ExpectedScheduledIndices map[string][]int
		// For each queue, indices of jobs expected to be preempted.
		// E.g., ExpectedPreemptedIndices["A"][0] is the indices of jobs declared for queue A in round 0.
		ExpectedPreemptedIndices map[string]map[int][]int
		// Expected market data
		// Will not be validated if no expectation is set (nil)
		ExpectedMarketData *MarketData
		// Indices of nodes on cordoned clusters.
		IndiciesOfNodesOnCordonedCluster []int
	}
	tests := map[string]struct {
		SchedulingConfig configuration.SchedulingConfig
		// Nodes to be considered by the scheduler.
		Nodes []*internaltypes.Node
		// Each item corresponds to a call to Reschedule().
		Rounds []SchedulingRound
		// Map from queue to the priority factor associated with that queue.
		PriorityFactorByQueue map[string]float64
		// Map of nodeId to jobs running on those nodes
		InitialRunningJobs map[int][]*jobdb.Job
	}{
		"three users, highest price jobs from single queue get on": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_C, 32),
						"C": testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_A, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"B": testfixtures.IntRange(0, 31),
					},
				},
				{
					// The system should be in steady-state; nothing should be scheduled/preempted.
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
						"C": testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_A, 32),
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1, "C": 1},
		},
		"three users, highest price jobs between queues get on": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 11),
							testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_A, 21)...,
						),
						"B": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_B, 11),
							testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_A, 21)...,
						),
						"C": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_B, 11),
							testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_A, 21)...,
						),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 10),
						"B": testfixtures.IntRange(0, 10),
						"C": testfixtures.IntRange(0, 9),
					},
				},
				{
					// The system should be in steady-state; nothing should be scheduled/preempted.
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_A, 21),
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_A, 21),
						"C": testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_A, 21),
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1, "C": 1},
		},
		"three users with same price - get even number of jobs": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_A, 21),
						),
						"B": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_A, 21),
						),
						"C": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_A, 21),
						),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 10),
						"B": testfixtures.IntRange(0, 10),
						"C": testfixtures.IntRange(0, 9),
					},
				},
				{
					// The system should be in steady-state; nothing should be scheduled/preempted.
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_A, 10),
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_A, 10),
						"C": testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_A, 11),
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1, "C": 1},
		},
		"Two users, no preemption if price lower": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 31),
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_A, 32),
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1},
		},
		"Two users, preemption if price higher": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 31),
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_C, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"B": testfixtures.IntRange(0, 31),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"A": {
							0: testfixtures.IntRange(0, 31),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1},
		},
		"Two users, partial preemption if price higher": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 31),
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"B": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_A, 16),
							testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_C, 16)...,
						),
					},
					ExpectedScheduledIndices: map[string][]int{
						"B": testfixtures.IntRange(16, 31),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"A": {
							0: testfixtures.IntRange(16, 31),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1},
		},
		"Self Preemption If Price Is Higher": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 31),
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": append(
							testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_A, 16),
							testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_C, 16)...,
						),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(16, 31),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"A": {
							0: testfixtures.IntRange(16, 31),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1},
		},
		"Two Users. Self preemption plus cross user preemption": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 31),
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_D, 16),
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_C, 32),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 15),
						"B": testfixtures.IntRange(0, 15),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"A": {
							0: testfixtures.IntRange(0, 31),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1},
		},
		"gang preemption": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 16),
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_B, 16),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 15),
						"B": testfixtures.IntRange(0, 15),
					},
				},
				{
					// Schedule a gang filling the remaining space on both nodes.
					JobsByQueue: map[string][]*jobdb.Job{
						"C": testfixtures.WithGangAnnotationsJobs(testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_C, 32)),
					},
					ExpectedScheduledIndices: map[string][]int{
						"C": testfixtures.IntRange(0, 31),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"A": {
							0: testfixtures.IntRange(0, 15),
						},
						"B": {
							0: testfixtures.IntRange(0, 15),
						},
					},
				},
				{
					// Schedule jobs that requires preempting one job in the gang,
					// and assert that all jobs in the gang are preempted.
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_D, 17),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 16),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"C": {
							1: testfixtures.IntRange(0, 31),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1, "C": 1},
		},
		"gang preempting high priced away jobs": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes: testfixtures.TestNodeFactory.AddTaints(testfixtures.N8GpuNodes(2, testfixtures.TestPriorities), []v1.Taint{
				{
					Key:    "gpu",
					Value:  "true",
					Effect: v1.TaintEffectNoSchedule,
				},
			}),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("A", bidstore.PriceBand_PRICE_BAND_D, testfixtures.PriorityClass4PreemptibleAway, 128),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 127),
					},
				},
				{
					// Schedule a gang filling the remaining space on both node
					// Queue A is preeempted despite having a higher price, because the jobs are scheduled as away jobs
					JobsByQueue: map[string][]*jobdb.Job{
						"B": testfixtures.N1GpuJobsWithPriceBandAndPriorityClass("B", bidstore.PriceBand_PRICE_BAND_A, testfixtures.PriorityClass6Preemptible, 16),
					},
					ExpectedScheduledIndices: map[string][]int{
						"B": testfixtures.IntRange(0, 15),
					},
					ExpectedPreemptedIndices: map[string]map[int][]int{
						"A": {
							0: testfixtures.IntRange(0, 127),
						},
					},
				},
				{
					// Queue A jobs don't schedule despite having a higher price, due to being away jobs
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("A", bidstore.PriceBand_PRICE_BAND_D, testfixtures.PriorityClass4PreemptibleAway, 128),
					},
					ExpectedScheduledIndices: map[string][]int{},
					ExpectedPreemptedIndices: map[string]map[int][]int{},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1, "C": 1},
		},
		"spot price - single queue": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 10),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 9),
					},
					ExpectedMarketData: &MarketData{
						ExpectedSpotPrice: 0,
						ExpectedBillableResource: map[string]internaltypes.ResourceList{
							// No spot price as price only set once 90% is scheduled
							"A": {},
						},
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_B, 30),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 21),
					},
					ExpectedMarketData: &MarketData{
						ExpectedSpotPrice: 2,
						ExpectedBillableResource: map[string]internaltypes.ResourceList{
							// Price set once 90% is scheduled
							// 29 jobs scheduled to reach 90%
							"A": testfixtures.CpuMem("29", "116Gi"),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1},
		},
		"spot price - multiple queues": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes:            testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBand("A", bidstore.PriceBand_PRICE_BAND_A, 10),
						"B": testfixtures.N1Cpu4GiJobsWithPriceBand("B", bidstore.PriceBand_PRICE_BAND_B, 10),
						"C": testfixtures.N1Cpu4GiJobsWithPriceBand("C", bidstore.PriceBand_PRICE_BAND_C, 20),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 1),
						"B": testfixtures.IntRange(0, 9),
						"C": testfixtures.IntRange(0, 19),
					},
					ExpectedMarketData: &MarketData{
						// Market price calculated at job that makes the pool 90% full
						// In this case it'll be the 29th job, which is queue B (price band B)
						ExpectedSpotPrice: 2,
						ExpectedBillableResource: map[string]internaltypes.ResourceList{
							// Queue A is not charged as it only has jobs scheduled after the spot price cutoff
							"A": testfixtures.TestResourceListFactory.MakeAllZero(),
							"B": testfixtures.CpuMem("9", "36Gi"),
							"C": testfixtures.CpuMem("20", "80Gi"),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1, "C": 1},
		},
		"preempted away jobs do not contribute to billable resource": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes: testfixtures.TestNodeFactory.AddTaints(testfixtures.N8GpuNodes(1, testfixtures.TestPriorities), []v1.Taint{
				{
					Key:    "gpu",
					Value:  "true",
					Effect: v1.TaintEffectNoSchedule,
				},
			}),
			Rounds: []SchedulingRound{
				{
					// A schedules away jobs first as it has the higher price band
					// B will preempt these with urgency based preemption
					// A should not get charged for having had resource scheduled at the time the spot price was calculcated
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("A", bidstore.PriceBand_PRICE_BAND_D, testfixtures.PriorityClass4PreemptibleAway, 1),
						"B": testfixtures.N1GpuJobsWithPriceBandAndPriorityClass("B", bidstore.PriceBand_PRICE_BAND_A, testfixtures.PriorityClass6Preemptible, 8),
					},
					ExpectedScheduledIndices: map[string][]int{
						"B": testfixtures.IntRange(0, 7),
					},
					ExpectedMarketData: &MarketData{
						ExpectedSpotPrice: 1,
						ExpectedBillableResource: map[string]internaltypes.ResourceList{
							"A": testfixtures.TestResourceListFactory.MakeAllZero(),
							"B": testfixtures.CpuMemGpu("64", "1024Gi", "8"),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1},
		},
		"jobs on cordoned clusters are not billable and do not contribute to spot price": {
			SchedulingConfig: testfixtures.WithMarketBasedSchedulingEnabled(testfixtures.TestSchedulingConfig()),
			Nodes: armadaslices.Concatenate(
				testfixtures.TestNodeFactory.AddLabels(testfixtures.N32CpuNodes(99, testfixtures.TestPriorities), map[string]string{"special": "true"}),
				testfixtures.N32CpuNodes(1, testfixtures.TestPriorities),
			),
			Rounds: []SchedulingRound{
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("A", bidstore.PriceBand_PRICE_BAND_A, testfixtures.PriorityClass6Preemptible, 32),
						"B": testfixtures.WithNodeSelectorJobs(map[string]string{"special": "true"},
							testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("B", bidstore.PriceBand_PRICE_BAND_D, testfixtures.PriorityClass6Preemptible, 3168)),
					},
					ExpectedScheduledIndices: map[string][]int{
						"A": testfixtures.IntRange(0, 31),
						"B": testfixtures.IntRange(0, 3167),
					},
					ExpectedMarketData: &MarketData{
						ExpectedSpotPrice: 4,
						ExpectedBillableResource: map[string]internaltypes.ResourceList{
							"A": testfixtures.TestResourceListFactory.MakeAllZero(),
							"B": testfixtures.CpuMem("2881", "11524Gi"),
						},
					},
				},
				{
					JobsByQueue: map[string][]*jobdb.Job{
						"A": testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("A", bidstore.PriceBand_PRICE_BAND_A, testfixtures.PriorityClass6Preemptible, 32),
						"B": testfixtures.WithNodeSelectorJobs(map[string]string{"special": "true"},
							testfixtures.N1Cpu4GiJobsWithPriceBandAndPriorityClass("B", bidstore.PriceBand_PRICE_BAND_D, testfixtures.PriorityClass6Preemptible, 3168)),
					},
					IndiciesOfNodesOnCordonedCluster: makeIntArray(99),
					ExpectedScheduledIndices:         map[string][]int{},
					ExpectedMarketData: &MarketData{
						ExpectedSpotPrice: 1,
						ExpectedBillableResource: map[string]internaltypes.ResourceList{
							// Price set once 90% is scheduled
							// 29 jobs scheduled to reach 90%
							"A": testfixtures.CpuMem("29", "116Gi"),
						},
					},
				},
			},
			PriorityFactorByQueue: map[string]float64{"A": 1, "B": 1},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			priorities := types.AllowedPriorities(tc.SchedulingConfig.PriorityClasses)

			jobDb := jobdb.NewJobDb(tc.SchedulingConfig.PriorityClasses, tc.SchedulingConfig.DefaultPriorityClassName, stringinterner.New(1024), testfixtures.TestResourceListFactory)
			jobDbTxn := jobDb.WriteTxn()

			// Add all the initial jobs, creating runs for them
			for nodeIdx, jobs := range tc.InitialRunningJobs {
				node := tc.Nodes[nodeIdx]
				for _, job := range jobs {
					err := jobDbTxn.Upsert([]*jobdb.Job{
						job.WithQueued(false).
							WithNewRun(node.GetExecutor(), node.GetId(), node.GetName(), node.GetPool(), job.PriorityClass().Priority),
					})
					require.NoError(t, err)
				}
			}

			// Accounting across scheduling rounds.
			roundByJobId := make(map[string]int)
			indexByJobId := make(map[string]int)
			allocatedByQueueAndPriorityClass := make(map[string]map[string]internaltypes.ResourceList)
			nodeIdByJobId := make(map[string]string)

			// Scheduling rate-limiters persist between rounds.
			// We control the rate at which time passes between scheduling rounds.
			schedulingStarted := time.Now()
			schedulingInterval := time.Second
			limiter := rate.NewLimiter(
				rate.Limit(tc.SchedulingConfig.MaximumSchedulingRate),
				tc.SchedulingConfig.MaximumSchedulingBurst,
			)
			limiterByQueue := make(map[string]*rate.Limiter)
			for queue := range tc.PriorityFactorByQueue {
				limiterByQueue[queue] = rate.NewLimiter(
					rate.Limit(tc.SchedulingConfig.MaximumPerQueueSchedulingRate),
					tc.SchedulingConfig.MaximumPerQueueSchedulingBurst,
				)
			}

			demandByQueue := map[string]internaltypes.ResourceList{}

			// Run the scheduler.
			ctx := armadacontext.Background()
			for i, round := range tc.Rounds {
				ctx = armadacontext.WithLogField(ctx, "round", i)
				ctx.Infof("starting scheduling round %d", i)

				jobsByNodeId := map[string][]*jobdb.Job{}
				for _, job := range jobDbTxn.GetAll() {
					if job.LatestRun() != nil && !job.LatestRun().InTerminalState() {
						nodeId := job.LatestRun().NodeId()
						jobsByNodeId[nodeId] = append(jobsByNodeId[nodeId], job)
					}
				}

				nodeDb, err := NewNodeDb(tc.SchedulingConfig, stringinterner.New(1024))
				require.NoError(t, err)
				nodeDbTxn := nodeDb.Txn(true)
				for index, node := range tc.Nodes {
					if !slices.Contains(round.IndiciesOfNodesOnCordonedCluster, index) {
						// Nodes on cordoned clusters do not get added to the node db
						err = nodeDb.CreateAndInsertWithJobDbJobsWithTxn(nodeDbTxn, jobsByNodeId[node.GetId()], node.DeepCopyNilKeys())
						require.NoError(t, err)
					}
				}
				nodeDbTxn.Commit()

				// Enqueue jobs that should be considered in this round.
				var queuedJobs []*jobdb.Job
				for queue, jobs := range round.JobsByQueue {
					for j, job := range jobs {
						job = job.WithQueued(true)
						require.Equal(t, queue, job.Queue())
						queuedJobs = append(queuedJobs, job.WithQueued(true))
						roundByJobId[job.Id()] = i
						indexByJobId[job.Id()] = j
						demandByQueue[job.Queue()] = demandByQueue[job.Queue()].Add(job.AllResourceRequirements())
					}
				}
				err = jobDbTxn.Upsert(queuedJobs)
				require.NoError(t, err)

				// If not provided, set total resources equal to the aggregate over tc.Nodes.
				totalResources := nodeDb.TotalKubernetesResources()

				fairnessCostProvider, err := fairness.NewDominantResourceFairness(
					nodeDb.TotalKubernetesResources(),
					testfixtures.TestPool,
					tc.SchedulingConfig,
				)
				require.NoError(t, err)
				sctx := context.NewSchedulingContext(
					testfixtures.TestPool,
					fairnessCostProvider,
					limiter,
					totalResources,
				)
				sctx.Started = schedulingStarted.Add(time.Duration(i) * schedulingInterval)

				for queue, priorityFactor := range tc.PriorityFactorByQueue {
					weight := 1 / priorityFactor
					queueDemand := demandByQueue[queue]
					err := sctx.AddQueueSchedulingContext(
						queue,
						weight,
						weight,
						allocatedByQueueAndPriorityClass[queue],
						queueDemand,
						queueDemand,
						internaltypes.ResourceList{},
						limiterByQueue[queue],
					)
					require.NoError(t, err)
				}
				constraints := schedulerconstraints.NewSchedulingConstraints(
					"pool",
					totalResources,
					tc.SchedulingConfig,
					armadaslices.Map(
						maps.Keys(tc.PriorityFactorByQueue),
						func(qn string) *api.Queue { return &api.Queue{Name: qn} },
					))
				sctx.UpdateFairShares()
				sch := NewPreemptingQueueScheduler(
					sctx,
					constraints,
					testfixtures.TestEmptyFloatingResources,
					tc.SchedulingConfig,
					jobDbTxn,
					nodeDb,
					false,
				)

				result, err := sch.Schedule(ctx)
				require.NoError(t, err)

				// Test resource accounting.
				for _, jctx := range result.PreemptedJobs {
					job := jctx.Job
					m := allocatedByQueueAndPriorityClass[job.Queue()]
					if m == nil {
						m = make(map[string]internaltypes.ResourceList)
						allocatedByQueueAndPriorityClass[job.Queue()] = m
					}
					m[job.PriorityClassName()] = m[job.PriorityClassName()].Subtract(job.AllResourceRequirements())
				}
				for _, jctx := range result.ScheduledJobs {
					job := jctx.Job
					m := allocatedByQueueAndPriorityClass[job.Queue()]
					if m == nil {
						m = make(map[string]internaltypes.ResourceList)
						allocatedByQueueAndPriorityClass[job.Queue()] = m
					}
					m[job.PriorityClassName()] = m[job.PriorityClassName()].Add(job.AllResourceRequirements())
				}
				for queue, qctx := range sctx.QueueSchedulingContexts {
					m := allocatedByQueueAndPriorityClass[queue]
					assert.Equal(t, internaltypes.RlMapRemoveZeros(m), internaltypes.RlMapRemoveZeros(qctx.AllocatedByPriorityClass))
				}

				// Test that jobs are mapped to nodes correctly.
				for _, jctx := range result.PreemptedJobs {
					job := jctx.Job
					nodeId := jctx.AssignedNode.GetId()
					assert.NotEmpty(t, nodeId)

					// Check that preempted jobs are preempted from the node they were previously scheduled onto.
					expectedNodeId := nodeIdByJobId[job.Id()]
					assert.Equal(t, expectedNodeId, nodeId, "job %s preempted from unexpected node", job.Id())
				}
				for _, jctx := range result.ScheduledJobs {
					job := jctx.Job
					nodeId := jctx.PodSchedulingContext.NodeId
					assert.NotEmpty(t, nodeId)

					node, err := nodeDb.GetNode(nodeId)
					require.NoError(t, err)
					assert.NotEmpty(t, node)

					// Check that the job can actually go onto this node.
					matches, reason, err := nodedb.StaticJobRequirementsMet(node, jctx)
					require.NoError(t, err)
					assert.Empty(t, reason)
					assert.True(t, matches)

					// Check that scheduled jobs are consistently assigned to the same node.
					// (We don't allow moving jobs between nodes.)
					if expectedNodeId, ok := nodeIdByJobId[job.Id()]; ok {
						assert.Equal(t, expectedNodeId, nodeId, "job %s scheduled onto unexpected node", job.Id())
					} else {
						nodeIdByJobId[job.Id()] = nodeId
					}
				}

				// Expected scheduled jobs.
				jobIdsByQueue := jobIdsByQueueFromJobContexts(result.ScheduledJobs)
				scheduledQueues := armadamaps.MapValues(round.ExpectedScheduledIndices, func(v []int) bool { return true })
				maps.Copy(scheduledQueues, armadamaps.MapValues(jobIdsByQueue, func(v []string) bool { return true }))
				for queue := range scheduledQueues {
					expected := round.ExpectedScheduledIndices[queue]
					jobIds := jobIdsByQueue[queue]
					actual := make([]int, 0)
					for _, jobId := range jobIds {
						actual = append(actual, indexByJobId[jobId])
					}
					slices.Sort(actual)
					slices.Sort(expected)
					assert.Equal(t, expected, actual, "scheduling from queue %s", queue)
				}

				// Expected preempted jobs.
				jobIdsByQueue = jobIdsByQueueFromJobContexts(result.PreemptedJobs)
				preemptedQueues := armadamaps.MapValues(round.ExpectedPreemptedIndices, func(v map[int][]int) bool { return true })
				maps.Copy(preemptedQueues, armadamaps.MapValues(jobIdsByQueue, func(v []string) bool { return true }))
				for queue := range preemptedQueues {
					expected := round.ExpectedPreemptedIndices[queue]
					jobIds := jobIdsByQueue[queue]
					actual := make(map[int][]int)
					for _, jobId := range jobIds {
						i := roundByJobId[jobId]
						j := indexByJobId[jobId]
						actual[i] = append(actual[i], j)
					}
					for _, s := range expected {
						slices.Sort(s)
					}
					for _, s := range actual {
						slices.Sort(s)
					}
					assert.Equal(t, expected, actual, "preempting from queue %s", queue)
				}

				// Expected market data
				if round.ExpectedMarketData != nil {
					assert.Equal(t, round.ExpectedMarketData.ExpectedSpotPrice, sctx.GetSpotPrice())
					for queue, expectedBillableResource := range round.ExpectedMarketData.ExpectedBillableResource {
						qctx, exists := sctx.QueueSchedulingContexts[queue]
						assert.True(t, exists, fmt.Sprintf("queue context for %s expected to exist as it has an expected billable resource set", queue))
						assert.Equal(t, expectedBillableResource, qctx.GetBillableResource())
					}
				}

				// We expect there to be no oversubscribed nodes.
				it, err := nodedb.NewNodesIterator(nodeDb.Txn(false))
				require.NoError(t, err)
				for node := it.NextNode(); node != nil; node = it.NextNode() {
					for _, p := range priorities {
						for _, r := range node.AllocatableByPriority[p].GetResources() {
							assert.True(t, r.RawValue >= 0, "resource %s oversubscribed by %d on node %s", r.Name, r.RawValue, node.GetId())
						}
					}
				}

				err = jobDbTxn.BatchDelete(armadaslices.Map(queuedJobs, func(job *jobdb.Job) string { return job.Id() }))
				require.NoError(t, err)

				var preemptedJobs []*jobdb.Job
				for _, jctx := range result.PreemptedJobs {
					job := jctx.Job
					preemptedJobs = append(
						preemptedJobs,
						job.
							WithUpdatedRun(job.LatestRun().WithFailed(true)).
							WithQueued(false).
							WithFailed(true),
					)
				}
				err = jobDbTxn.Upsert(preemptedJobs)
				require.NoError(t, err)

				// Jobs may arrive out of order here; sort them, so that runs
				// are created in the right order (this influences the order in
				// which jobs are preempted).
				slices.SortFunc(
					result.ScheduledJobs,
					func(a, b *context.JobSchedulingContext) int {
						if a.Job.SubmitTime().Before(b.Job.SubmitTime()) {
							return -1
						} else if b.Job.SubmitTime().Before(a.Job.SubmitTime()) {
							return 1
						} else {
							return 0
						}
					},
				)
				var scheduledJobs []*jobdb.Job
				for _, jctx := range result.ScheduledJobs {
					job := jctx.Job
					jobId := job.Id()
					node, err := nodeDb.GetNode(jctx.PodSchedulingContext.NodeId)
					require.NotNil(t, node)
					require.NoError(t, err)
					priority, ok := nodeDb.GetScheduledAtPriority(jobId)
					require.True(t, ok)
					scheduledJobs = append(
						scheduledJobs,
						job.WithQueuedVersion(job.QueuedVersion()+1).
							WithQueued(false).
							WithNewRun(node.GetExecutor(), node.GetId(), node.GetName(), node.GetPool(), priority),
					)
				}
				err = jobDbTxn.Upsert(scheduledJobs)
				require.NoError(t, err)
			}
		})
	}
}

func makeIntArray(maxValue int) []int {
	result := []int{}
	for i := 0; i < maxValue; i++ {
		result = append(result, i)
	}
	return result
}
