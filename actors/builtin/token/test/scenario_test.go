package test

import (
	"bytes"
	"context"
	tutil "github.com/filecoin-project/specs-actors/v3/support/testing"
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
	holder2 := addrs[2]
	holder3 := addrs[3]
	holder4 := addrs[4]

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

	// founder gives tokens to first 2 users
	trnsfrParams := token.TransferParams{
		To:    holder1,
		Value: abi.NewTokenAmount(3000),
	}
	ret = vm.ApplyOk(t, v, founder, tokenActorAddr, big.Zero(), builtin.MethodsToken.Transfer, &trnsfrParams)
	assert.Nil(t, ret)

	trnsfrParams = token.TransferParams{
		To:    holder2,
		Value: abi.NewTokenAmount(4000),
	}
	ret = vm.ApplyOk(t, v, founder, tokenActorAddr, big.Zero(), builtin.MethodsToken.Transfer, &trnsfrParams)
	assert.Nil(t, ret)

	// confirm both accounts have balance
	ret = vm.ApplyOk(t, v, holder1, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &holder1)
	balance := ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(3000))

	ret = vm.ApplyOk(t, v, holder2, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &holder2)
	balance = ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(4000))

	// holder1 transfers to holder3
	trnsfrParams = token.TransferParams{
		To:    holder3,
		Value: abi.NewTokenAmount(1000),
	}
	ret = vm.ApplyOk(t, v, holder1, tokenActorAddr, big.Zero(), builtin.MethodsToken.Transfer, &trnsfrParams)
	assert.Nil(t, ret)

	ret = vm.ApplyOk(t, v, holder3, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &holder3)
	balance = ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(1000))

	// founder authorizes holder3 to make transfers
	approveParams := token.ApproveParams{
		Approvee: holder3,
		Value:    abi.NewTokenAmount(5000),
	}
	ret = vm.ApplyOk(t, v, founder, tokenActorAddr, big.Zero(), builtin.MethodsToken.Approve, &approveParams)
	assert.Nil(t, ret)

	allowanceParams := token.AllowanceParams{
		Owner:   founder,
		Spender: holder3,
	}
	ret = vm.ApplyOk(t, v, holder3, tokenActorAddr, big.Zero(), builtin.MethodsToken.Allowance, &allowanceParams)
	allowance := ret.(*abi.TokenAmount)
	assert.Equal(t, *allowance, abi.NewTokenAmount(5000))

	transferFromParams := token.TransferFromParams{
		From:  founder,
		To:    holder4,
		Value: abi.NewTokenAmount(3500),
	}
	ret = vm.ApplyOk(t, v, holder3, tokenActorAddr, big.Zero(), builtin.MethodsToken.TransferFrom, &transferFromParams)
	assert.Nil(t, ret)

	ret = vm.ApplyOk(t, v, holder4, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &holder4)
	balance = ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(3500))
}

func TestGiveToPublicKeyAddress(t *testing.T) {
	ctx := context.Background()
	v := vm.NewVMWithSingletons(ctx, t, ipld.NewBlockStoreInMemory())
	v, err := v.WithNetworkVersion(network.Version5)
	require.NoError(t, err)
	addrs := vm.CreateAccounts(ctx, t, v, 5, big.Mul(big.NewInt(10_000), big.NewInt(1e18)), 93837778)

	founder := addrs[0]

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

	holder1 := tutil.NewSECP256K1Addr(t, "secpaddress")

	// founder gives tokens to first 2 users
	trnsfrParams := token.TransferParams{
		To:    holder1,
		Value: abi.NewTokenAmount(3000),
	}
	ret = vm.ApplyOk(t, v, founder, tokenActorAddr, big.Zero(), builtin.MethodsToken.Transfer, &trnsfrParams)
	assert.Nil(t, ret)

	ret = vm.ApplyOk(t, v, holder1, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &holder1)
	balance := ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(3000))
}
