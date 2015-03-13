package assert

import (
	"github.com/cloudfoundry/cli/cf/errors"
)

func Condition(cond bool, message string) {
	if !cond {
		panic(errors.New("PLUGIN ERROR: " + message))
	}
}

func ErrorIsNil(err error) {
	if err != nil {
		Condition(false, "error not nil, "+err.Error())
	}
}
