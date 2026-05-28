package output

import (
	"errors"

	"github.com/rahadiangg/bbx/internal/fail"
)

func asFailError(err error) (*fail.Error, bool) {
	var fe *fail.Error
	if errors.As(err, &fe) {
		return fe, true
	}
	return nil, false
}
