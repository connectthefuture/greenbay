package main

import (
	"flag"
	"testing"

	"github.com/mongodb/grip"
	"github.com/mongodb/grip/level"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

// MainSuite is a collection of tests that exercise the main() of the
// program, and associated operations and top-level configuration.
type MainSuite struct {
	suite.Suite
}

func TestMainSuite(t *testing.T) {
	suite.Run(t, new(MainSuite))
}

func (s *MainSuite) TestLoggingSetupUsingDefaultSender() {
	grip.SetName("foo")
	s.Equal(grip.Name(), "foo")

	loggingSetup("test", "info")
	s.Equal(grip.Name(), "test")
}

func (s *MainSuite) TestLogSetupWithInvalidLevelDoesNotChangeLevel() {
	// when you specify an invalid level, grip shouldn't change
	// the level.
	s.Equal(grip.ThresholdLevel(), level.Info)

	loggingSetup("test", "QUIET")
	s.Equal(grip.ThresholdLevel(), level.Info)

	// Following case is just to make sure that normal
	// setting still works as expected.
	loggingSetup("test", "debug")
	s.Equal(grip.ThresholdLevel(), level.Debug)
}

func (s *MainSuite) TestAppBuilderFunctionSetsCorrectProperties() {
	app := buildApp()

	s.Equal("greenbay", app.Name)

	// the exact number will change, but should be >0
	s.NotEqual(len(app.Commands), 0)

	// The app should have some top level flags, and the first
	// flag should be the logging-level configuration.
	s.NotZero(app.Flags)
	s.Equal(app.Flags[0].GetName(), "level")

	// we do logging set up here, so it needs to be set
	s.NotZero(app.Before)

	s.NoError(app.Before(cli.NewContext(app, &flag.FlagSet{}, nil)))
}

func (s *MainSuite) TestChecksActionFunctionReturnsErrorWithoutArguments() {
	cmd := checks()
	ctx := cli.NewContext(buildApp(), &flag.FlagSet{}, nil)
	checkFunc, ok := cmd.Action.(func(c *cli.Context) error)
	s.True(ok)
	err := checkFunc(ctx)
	s.Error(err)
}
