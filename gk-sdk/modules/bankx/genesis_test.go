package bankx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coinexchain/cet-sdk/modules/bankx"
	"github.com/coinexchain/cet-sdk/modules/bankx/internal/types"
)

func TestValidate(t *testing.T) {
	genes := bankx.DefaultGenesisState()
	err := genes.ValidateGenesis()
	require.Equal(t, nil, err)

	errGenes := bankx.NewGenesisState(bankx.NewParams(-1, 0, 0))
	require.Equal(t, errGenes.ValidateGenesis(), types.ErrInvalidActivatingFee())
	errGenes = bankx.NewGenesisState(bankx.NewParams(0, -1, 0))
	require.Equal(t, errGenes.ValidateGenesis(), types.ErrInvalidLockCoinsFreeTime())
	errGenes = bankx.NewGenesisState(bankx.NewParams(0, 0, -1))
	require.Equal(t, errGenes.ValidateGenesis(), types.ErrInvalidLockCoinsFee())
}

func TestInitGenesis(t *testing.T) {
	genes := bankx.DefaultGenesisState()
	bkx, _, ctx := defaultContext()
	bankx.InitGenesis(ctx, *bkx, genes)
	gen := bankx.ExportGenesis(ctx, *bkx)
	require.Equal(t, genes, gen)
}
