package provision

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/grimange/almadial/internal/artifacts"
	"github.com/grimange/almadial/internal/config"
	"github.com/grimange/almadial/internal/plan"
	"github.com/grimange/almadial/internal/preflight"
)

// Inputs preserves ALM-11's actual configuration/preflight identity along with
// S01's RPM observations. Lab qualification can target an existing disposable
// installation; that exception never makes its normal install plan eligible.
type Inputs struct {
	Config                 config.Config          `json:"config"`
	Preflight              preflight.Snapshot     `json:"preflight"`
	Requirements           preflight.Requirements `json:"requirements"`
	Inventory              Snapshot               `json:"inventory"`
	Transaction            *Transaction           `json:"transaction,omitempty"`
	PrepareRepositoryTrust bool                   `json:"prepare_repository_trust,omitempty"`
}

func (i Inputs) Prepare(m artifacts.Manifest) (Stage, error) {
	p, e := plan.New(i.Config, i.Preflight, i.Requirements, m.Digest())
	if e != nil {
		return Stage{}, e
	}
	// A forged or accidentally mixed parent observation is not a host binding.
	id, e := hex.DecodeString(i.Preflight.HostID)
	if e != nil || len(id) != 16 {
		return Stage{}, errPolicy
	}
	h := sha256.Sum256([]byte(i.Preflight.HostID))
	if hex.EncodeToString(h[:]) != i.Inventory.Host {
		return Stage{}, errPolicy
	}
	if i.PrepareRepositoryTrust {
		if i.Transaction != nil {
			return Stage{}, errPolicy
		}
		return PrepareRepositoryTrust(p, m, i.Inventory)
	}
	if i.Transaction != nil {
		return PreparePackages(p, m, i.Inventory, *i.Transaction)
	}
	return PrepareResolution(p, m, i.Inventory)
}
