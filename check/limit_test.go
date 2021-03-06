package check

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitCheckImplementation(t *testing.T) {
	assert := assert.New(t)

	check := &limitCheck{
		Base: NewBase("limit", 0),
		limitTest: func(_ int) (bool, error) {
			return true, errors.New("a")
		},
	}

	check.Run()
	assert.Error(check.Error())
	assert.False(check.Output().Passed)

	check = &limitCheck{
		Base: NewBase("limit", 0),
		limitTest: func(_ int) (bool, error) {
			return false, nil
		},
	}

	check.Run()
	assert.Error(check.Error())
	assert.False(check.Output().Passed)
}
