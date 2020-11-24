package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/params"
)

// Default parameter values
const (
	DefaultMinSelfDelegation = int64(10000e8)
)

// Parameter keys
var (
	KeyMinSelfDelegation          = []byte("MinSelfDelegation")
	KeyMinMandatoryCommissionRate = []byte("MinMandatoryCommissionRate")

	DefaultMinMandatoryCommissionRate = sdk.NewDecWithPrec(1, 1)
)

var _ params.ParamSet = (*Params)(nil)

// Params defines the parameters for the stakingx module.
type Params struct {
	MinSelfDelegation          int64   `json:"min_self_delegation"`
	MinMandatoryCommissionRate sdk.Dec `json:"min_mandatory_commission_rate"`
}

// ParamKeyTable for stakingx module
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return Params{
		MinSelfDelegation:          DefaultMinSelfDelegation,
		MinMandatoryCommissionRate: DefaultMinMandatoryCommissionRate,
	}
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of stakingx module's parameters.
func (p *Params) ParamSetPairs() params.ParamSetPairs {
	return params.ParamSetPairs{
		{Key: KeyMinSelfDelegation, Value: &p.MinSelfDelegation},
		{Key: KeyMinMandatoryCommissionRate, Value: &p.MinMandatoryCommissionRate},
	}
}
