package jobdb

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	clock "k8s.io/utils/clock/testing"

	"github.com/armadaproject/armada/internal/common/stringinterner"
	"github.com/armadaproject/armada/internal/common/types"
	"github.com/armadaproject/armada/internal/scheduler/internaltypes"
)

const (
	PriorityClass0               = "priority-0"
	PriorityClass1               = "priority-1"
	PriorityClass2               = "priority-2"
	PriorityClass2NonPreemptible = "priority-2-non-preemptible"
	PriorityClass3               = "priority-3"
)

var (
	TestPriorityClasses = map[string]types.PriorityClass{
		PriorityClass0:               {Priority: 0, Preemptible: true},
		PriorityClass1:               {Priority: 1, Preemptible: true},
		PriorityClass2:               {Priority: 2, Preemptible: true},
		PriorityClass2NonPreemptible: {Priority: 2, Preemptible: false},
		PriorityClass3:               {Priority: 3, Preemptible: false},
	}
	TestDefaultPriorityClass = PriorityClass3
	SchedulingKeyGenerator   = internaltypes.NewSchedulingKeyGeneratorWithKey(make([]byte, 32))
	testClock                = clock.NewFakeClock(time.Now())
	jobDb                    = NewJobDbWithSchedulingKeyGenerator(
		TestPriorityClasses,
		TestDefaultPriorityClass,
		SchedulingKeyGenerator,
		stringinterner.New(1024),
		makeTestResourceListFactory(),
	)
	scheduledAtPriority = int32(5)
)

func init() {
	jobDb.clock = testClock
}

var baseJobRun = jobDb.CreateRun(
	uuid.New().String(),
	uuid.NewString(),
	5,
	"test-executor",
	"test-nodeId",
	"test-nodeName",
	"test-pool",
	&scheduledAtPriority,
	false,
	false,
	false,
	false,
	false,
	false,
	false,
	false,
	nil,
	nil,
	nil,
	nil,
	nil,
	false,
	false,
)

// Test methods that only have getters
func TestJobRun_TestGetter(t *testing.T) {
	assert.Equal(t, baseJobRun.id, baseJobRun.Id())
	assert.Equal(t, baseJobRun.created, baseJobRun.Created())
	assert.Equal(t, baseJobRun.executor, baseJobRun.Executor())
	assert.Equal(t, baseJobRun.nodeId, baseJobRun.NodeId())
}

func TestJobRun_TestRunning(t *testing.T) {
	runningRun := baseJobRun.WithRunning(true)
	assert.False(t, baseJobRun.Running())
	assert.True(t, runningRun.Running())
}

func TestJobRun_TestSucceeded(t *testing.T) {
	succeededRun := baseJobRun.WithSucceeded(true)
	assert.False(t, baseJobRun.Succeeded())
	assert.True(t, succeededRun.Succeeded())
}

func TestJobRun_TestFailed(t *testing.T) {
	failedRun := baseJobRun.WithFailed(true)
	assert.False(t, baseJobRun.Failed())
	assert.True(t, failedRun.Failed())
}

func TestJobRun_TestCancelled(t *testing.T) {
	cancelledRun := baseJobRun.WithCancelled(true)
	assert.False(t, baseJobRun.Cancelled())
	assert.True(t, cancelledRun.Cancelled())
}

func TestJobRun_TestPreemptRequested(t *testing.T) {
	preemptRequestedRun := baseJobRun.WithPreemptRequested(true)
	assert.False(t, baseJobRun.PreemptRequested())
	assert.True(t, preemptRequestedRun.PreemptRequested())
}

func TestJobRun_TestReturned(t *testing.T) {
	returnedRun := baseJobRun.WithReturned(true)
	assert.False(t, baseJobRun.Returned())
	assert.True(t, returnedRun.Returned())
}

func TestJobRun_TestRunAttempted(t *testing.T) {
	attemptedRun := baseJobRun.WithAttempted(true)
	assert.False(t, baseJobRun.RunAttempted())
	assert.True(t, attemptedRun.RunAttempted())
}

func TestDeepCopy(t *testing.T) {
	run := jobDb.CreateRun(
		uuid.NewString(),
		"job id",
		1,
		"executor",
		"nodeName",
		"nodeId",
		"pool",
		&scheduledAtPriority,
		true,
		true,
		true,
		true,
		true,
		true,
		true,
		true,
		nil,
		nil,
		nil,
		nil,
		nil,
		true,
		true,
	)
	expected := jobDb.CreateRun(
		run.id,
		"job id",
		1,
		"executor",
		"nodeName",
		"nodeId",
		"pool",
		&scheduledAtPriority,
		true,
		true,
		true,
		true,
		true,
		true,
		true,
		true,
		nil,
		nil,
		nil,
		nil,
		nil,
		true,
		true,
	)
	actual := run.DeepCopy()
	run.nodeId = "new nodeId"
	run.executor = "new executor"
	assert.Equal(t, expected, actual)
}
