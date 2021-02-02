package test

import (
	"bytes"
	"context"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	init_ "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/support/ipld"
	vm "github.com/filecoin-project/specs-actors/v3/support/vm"
)

func TestCreateAndUseToken(t *testing.T) {
	ctx := context.Background()
	v := vm.NewVMWithSingletons(ctx, t, ipld.NewBlockStoreInMemory())
	v, err := v.WithNetworkVersion(network.Version5)
	require.NoError(t, err)
	addrs := vm.CreateAccounts(ctx, t, v, 5, big.Mul(big.NewInt(10_000), big.NewInt(1e18)), 93837778)

	founder := addrs[0]
	holder1 := addrs[1]
	//holder2 := addrs[2]
	//holder3 := addrs[3]
	//holder4 := addrs[4]

	tokenParams := token.ConstructorParams{
		Name:          "TestCoin",
		Symbol:        "TCN",
		Icon:          []byte("testcoin icon"),
		Decimals:      5,
		TotalSupply:   abi.NewTokenAmount(1e12),
		SystemAccount: founder,
	}

	paramBuf := new(bytes.Buffer)
	err = tokenParams.MarshalCBOR(paramBuf)
	require.NoError(t, err)

	initParam := init_.ExecParams{
		CodeCID:           builtin.TokenActorCodeID,
		ConstructorParams: paramBuf.Bytes(),
	}
	ret := vm.ApplyOk(t, v, founder, builtin.InitActorAddr, big.Zero(), builtin.MethodsInit.Exec, &initParam)
	initRet := ret.(*init_.ExecReturn)
	assert.NotNil(t, initRet)
	tokenActorAddr := initRet.IDAddress

	// founder gives 3000 tokens to first 2 users
	trnsfrParams := token.TransferParams{
		To:    holder1,
		Value: abi.NewTokenAmount(3000),
	}
	ret = vm.ApplyOk(t, v, founder, tokenActorAddr, big.Zero(), builtin.MethodsToken.Transfer, &trnsfrParams)
	assert.Nil(t, ret)

	//removeParams := multisig.RemoveSignerParams{
	//	Signer:   addrs[0],
	//	Decrease: false,
	//}
	//
	//paramBuf = new(bytes.Buffer)
	//err = removeParams.MarshalCBOR(paramBuf)
	//require.NoError(t, err)
	//
	//proposeRemoveSignerParams := multisig.ProposeParams{
	//	To:     multisigAddr,
	//	Value:  big.Zero(),
	//	Method: builtin.MethodsMultisig.RemoveSigner,
	//	Params: paramBuf.Bytes(),
	//}
	//// address 0 fails when trying to execute the transaction removing address 0
	//_, code := v.ApplyMessage(addrs[0], multisigAddr, big.Zero(), builtin.MethodsMultisig.Propose, &proposeRemoveSignerParams)
	//assert.Equal(t, exitcode.ErrIllegalState, code)
	//// address 1 succeeds when trying to execute the transaction removing address 0
	//vm.ApplyOk(t, v, addrs[1], multisigAddr, big.Zero(), builtin.MethodsMultisig.Propose, &proposeRemoveSignerParams)
}
