package extensionContracts

import (
	"github.com/kisexp/xdchain/core/state"
)

type AccountWithMetadata struct {
	State state.DumpAccount `json:"state"`
}
