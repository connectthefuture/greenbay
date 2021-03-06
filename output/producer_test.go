package output

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/job"
	"github.com/mongodb/amboy/queue"
	"github.com/mongodb/greenbay/check"
	"github.com/mongodb/grip"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type ProducerSuite struct {
	tmpDir   string
	results  ResultsProducer
	factory  ResultsFactory
	require  *require.Assertions
	cancel   context.CancelFunc
	queue    amboy.Queue
	isGoTest bool
	suite.Suite
}

// Constructors. Run this suite of tests for every ResultsProducer
// implementation.

func TestGoTestProducerSuite(t *testing.T) {
	s := new(ProducerSuite)
	s.isGoTest = true
	s.factory = func() ResultsProducer {
		return &GoTest{
			buf: bytes.NewBuffer([]byte{}),
		}
	}

	suite.Run(t, s)
}

func TestResultsProducerSuite(t *testing.T) {
	s := new(ProducerSuite)
	s.factory = func() ResultsProducer {
		return &Results{}
	}

	suite.Run(t, s)
}

func TestGripProducerSuite(t *testing.T) {
	s := new(ProducerSuite)
	s.factory = func() ResultsProducer {
		return &GripOutput{}
	}

	suite.Run(t, s)
}

// Fixtures for suite:

func (s *ProducerSuite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.require = s.Require()
	s.queue = queue.NewLocalUnordered(2)
	s.require.NoError(s.queue.Start(ctx))
	tmpDir, err := ioutil.TempDir("", uuid.NewV4().String())
	s.require.NoError(err)
	s.tmpDir = tmpDir
	for i := 0; i < 10; i++ {
		check := &mockCheck{Base: check.Base{Base: &job.Base{}}}
		check.SetID(fmt.Sprintf("mock-check-%d", i))
		if i%3 == 0 {
			check.Base.Message = fmt.Sprintf("count=%d", i)
		}

		if i%2 == 0 {
			check.Base.Errors = []string{"even"}
		}

		s.NoError(s.queue.Put(check))
	}

	amboy.Wait(s.queue)
}

func (s *ProducerSuite) SetupTest() {
	s.results = s.factory()
}

func (s *ProducerSuite) TearDownSuite() {
	s.cancel()
	s.require.NoError(os.RemoveAll(s.tmpDir))
}

// Test cases:

func (s *ProducerSuite) TestPopulateOperationDoNotErrorWithBasicTasks() {
	s.NoError(s.results.Populate(s.queue.Results()))
}

func (s *ProducerSuite) TestOutputMethodsFailIfJobsHaveErrors() {

	// everything is a pointer inside the queue so this should work:
	for t := range s.queue.Results() {
		task := t.(*mockCheck)
		task.Base.WasSuccessful = false
	}

	s.NoError(s.results.Populate(s.queue.Results()))

	s.Error(s.results.ToFile(filepath.Join(s.tmpDir, "one")))

	for t := range s.queue.Results() {
		task := t.(*mockCheck)
		task.Base.WasSuccessful = true
	}
}

func (s *ProducerSuite) TestPrintMethodReturnsNoErrorIfAllOperationsAreSuccessful() {
	s.NoError(s.results.Populate(s.queue.Results()))

	if s.isGoTest {
		s.Suite.T().Skip("skipping printing results for go test because it is confusing")
	}

	grip.Alert("printing test results")
	s.NoError(s.results.Print())
	grip.Alert("completed printing results")
}

func (s *ProducerSuite) TestToFileMethodReturnsNoErrorIfAllOperationsAreSuccessful() {
	s.NoError(s.results.Populate(s.queue.Results()))

	err := s.results.ToFile(filepath.Join(s.tmpDir, "two"))
	grip.Error(err)
	s.NoError(err)
}

func (s *ProducerSuite) TestWithQueueAndInvalidJobs() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := queue.NewLocalUnordered(2)
	s.require.NoError(q.Start(ctx))

	s.NoError(q.Put(job.NewShellJob("echo foo", "")))
	amboy.Wait(q)
	s.Error(s.results.Populate(q.Results()))
}

func (s *ProducerSuite) TestToFileMethodShouldFailOnNonWriteableFiles() {
	s.NoError(s.results.Populate(s.queue.Results()))

	fn := filepath.Join(s.tmpDir, "foo", "three")
	_, err := os.Stat(fn)
	s.True(os.IsNotExist(err))

	err = s.results.ToFile(fn)
	s.Error(err)
	grip.Error(err)
}
