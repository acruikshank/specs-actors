package token_test

import (
	"context"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"
	tutil "github.com/filecoin-project/specs-actors/v3/support/testing"
	"testing"

	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/support/mock"
	"github.com/stretchr/testify/assert"
)

func TestExports(t *testing.T) {
	mock.CheckActorExports(t, token.Actor{})
}

func TestConstruction(t *testing.T) {
	actor := &tActorHarness{token.Actor{}, t}
	receiver := tutil.NewIDAddr(t, 100)
	system := tutil.NewIDAddr(t, 101)
	rt := mock.NewBuilder(context.Background(), receiver).
		WithCaller(builtin.InitActorAddr, builtin.InitActorCodeID).
		WithActorType(system, builtin.AccountActorCodeID).
		Build(t)

	// Create empty multisig
	rt.SetEpoch(100)
	name := "Testcoin"
	symbol := "TCN"
	icon := []byte("testcoin icon")
	decimals := uint64(5)
	supply := abi.NewTokenAmount(1e12)
	actor.constructAndVerify(rt, name, symbol, icon, decimals, supply, system)

	var st token.State
	rt.GetState(&st)
	info := st.TokenInfo()

	assert.Equal(t, info.Name, name)
	assert.Equal(t, info.Symbol, symbol)
	assert.Equal(t, info.Icon, icon)
	assert.Equal(t, info.Decimals, decimals)
	assert.Equal(t, info.TotalSupply, supply)
}

func TestBalanceOf(t *testing.T) {
	actor := &tActorHarness{token.Actor{}, t}
	receiver := tutil.NewIDAddr(t, 100)
	system := tutil.NewIDAddr(t, 101)
	rt := mock.NewBuilder(context.Background(), receiver).
		WithCaller(builtin.InitActorAddr, builtin.InitActorCodeID).
		WithActorType(system, builtin.AccountActorCodeID).
		Build(t)

	rt.SetEpoch(100)
	supply := abi.NewTokenAmount(1e12)
	actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

	balance := actor.balanceOf(rt, system)

	assert.Equal(t, supply, balance)
}

func TestTransfer(t *testing.T) {
	actor := &tActorHarness{token.Actor{}, t}
	receiver := tutil.NewIDAddr(t, 100)
	system := tutil.NewIDAddr(t, 101)
	user1 := tutil.NewIDAddr(t, 102)
	user2 := tutil.NewIDAddr(t, 103)
	builder := mock.NewBuilder(context.Background(), receiver).
		WithCaller(builtin.InitActorAddr, builtin.InitActorCodeID).
		WithActorType(system, builtin.AccountActorCodeID).
		WithActorType(user1, builtin.AccountActorCodeID).
		WithActorType(user2, builtin.AccountActorCodeID)

	t.Run("single transfer", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(5000)
		actor.transfer(rt, system, user1, trnsfrAmt)

		sysBalance := actor.balanceOf(rt, system)
		assert.Equal(t, big.Sub(supply, trnsfrAmt), sysBalance)

		userBalance := actor.balanceOf(rt, user1)
		assert.Equal(t, trnsfrAmt, userBalance)
	})

	t.Run("insufficient funds", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := big.Add(supply, abi.NewTokenAmount(1))

		rt.ExpectAbortContainsMessage(exitcode.ErrInsufficientFunds, "insufficient funds", func() {
			actor.transfer(rt, system, user1, trnsfrAmt)
		})
	})

	t.Run("muliple transfers", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(5000)
		actor.transfer(rt, system, user1, trnsfrAmt)

		trnsfrAmt2 := abi.NewTokenAmount(2000)
		actor.transfer(rt, user1, user2, trnsfrAmt2)

		sysBalance := actor.balanceOf(rt, system)
		assert.Equal(t, big.Sub(supply, trnsfrAmt), sysBalance)

		user1Balance := actor.balanceOf(rt, user1)
		assert.Equal(t, big.Sub(trnsfrAmt, trnsfrAmt2), user1Balance)

		user2Balance := actor.balanceOf(rt, user2)
		assert.Equal(t, trnsfrAmt2, user2Balance)

		// transfer remaining balance
		actor.transfer(rt, user1, user2, big.Sub(trnsfrAmt, trnsfrAmt2))

		user1Balance = actor.balanceOf(rt, user1)
		assert.Equal(t, big.Zero(), user1Balance)

		user2Balance = actor.balanceOf(rt, user2)
		assert.Equal(t, trnsfrAmt, user2Balance)
	})

	t.Run("transfer from user with no balance", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(2000)

		rt.ExpectAbortContainsMessage(exitcode.ErrInsufficientFunds, "insufficient funds", func() {
			actor.transfer(rt, user1, user2, trnsfrAmt)
		})
	})

	t.Run("insufficient funds", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := big.Add(supply, abi.NewTokenAmount(1))

		rt.ExpectAbortContainsMessage(exitcode.ErrInsufficientFunds, "insufficient funds", func() {
			actor.transfer(rt, system, user1, trnsfrAmt)
		})
	})
}

func TestApprovedTransfer(t *testing.T) {
	actor := &tActorHarness{token.Actor{}, t}
	receiver := tutil.NewIDAddr(t, 100)
	system := tutil.NewIDAddr(t, 101)
	user1 := tutil.NewIDAddr(t, 102)
	user2 := tutil.NewIDAddr(t, 103)
	user3 := tutil.NewIDAddr(t, 104)
	builder := mock.NewBuilder(context.Background(), receiver).
		WithCaller(builtin.InitActorAddr, builtin.InitActorCodeID).
		WithActorType(system, builtin.AccountActorCodeID).
		WithActorType(user1, builtin.AccountActorCodeID).
		WithActorType(user2, builtin.AccountActorCodeID).
		WithActorType(user3, builtin.AccountActorCodeID)

	t.Run("approve transferring", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(5000)
		actor.approve(rt, system, user1, trnsfrAmt)

		allowance := actor.allowance(rt, system, user1)
		assert.Equal(t, trnsfrAmt, allowance)
	})

	t.Run("transfer from without approval", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(5000)

		rt.ExpectAbortContainsMessage(exitcode.ErrForbidden, "insufficient allowance", func() {
			actor.transferFrom(rt, user1, system, user2, trnsfrAmt)
		})
	})

	t.Run("transfer approved amount", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(5000)
		actor.approve(rt, system, user1, trnsfrAmt)
		actor.transferFrom(rt, user1, system, user2, trnsfrAmt)

		user2Balance := actor.balanceOf(rt, user2)
		assert.Equal(t, trnsfrAmt, user2Balance)
	})

	t.Run("transfer exceeds allowance", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(5000)
		actor.approve(rt, system, user1, trnsfrAmt)

		rt.ExpectAbortContainsMessage(exitcode.ErrForbidden, "insufficient allowance", func() {
			actor.transferFrom(rt, user1, system, user2, big.Add(trnsfrAmt, abi.NewTokenAmount(1)))
		})
	})

	t.Run("transfer exceeds available funds", func(t *testing.T) {
		rt := builder.Build(t)
		rt.SetEpoch(100)
		supply := abi.NewTokenAmount(1e12)
		actor.constructAndVerify(rt, "Testcoin", "TCN", []byte("testcoin icon"), uint64(5), supply, system)

		trnsfrAmt := abi.NewTokenAmount(1e13)
		actor.approve(rt, system, user1, trnsfrAmt)

		rt.ExpectAbortContainsMessage(exitcode.ErrInsufficientFunds, "insufficient funds", func() {
			actor.transferFrom(rt, user1, system, user2, big.Add(supply, abi.NewTokenAmount(1)))
		})
	})
}

type tActorHarness struct {
	a token.Actor
	t testing.TB
}

func (h *tActorHarness) constructAndVerify(rt *mock.Runtime, name string, symbol string, icon []byte, decimals uint64, supply abi.TokenAmount, system addr.Address) {
	constructParams := token.ConstructorParams{
		Name:          name,
		Symbol:        symbol,
		Icon:          icon,
		Decimals:      decimals,
		TotalSupply:   supply,
		SystemAccount: system,
	}

	rt.ExpectValidateCallerAddr(builtin.InitActorAddr)
	ret := rt.Call(h.a.Constructor, &constructParams)
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tActorHarness) transfer(rt *mock.Runtime, from, to addr.Address, value abi.TokenAmount) {
	transferParams := token.TransferParams{
		To:    to,
		Value: value,
	}

	rt.ExpectValidateCallerAny()
	rt.SetCaller(from, builtin.AccountActorCodeID)
	ret := rt.Call(h.a.Transfer, &transferParams)
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tActorHarness) balanceOf(rt *mock.Runtime, acct addr.Address) abi.TokenAmount {
	rt.ExpectValidateCallerAny()
	ret := rt.Call(h.a.BalanceOf, &acct)
	rt.Verify()

	balance := ret.(*abi.TokenAmount)
	return *balance
}

func (h *tActorHarness) approve(rt *mock.Runtime, approver, approvee addr.Address, value abi.TokenAmount) {
	approveParameters := token.ApproveParams{
		Approvee: approvee,
		Value:    value,
	}

	rt.ExpectValidateCallerAny()
	rt.SetCaller(approver, builtin.AccountActorCodeID)
	ret := rt.Call(h.a.Approve, &approveParameters)
	assert.Nil(h.t, ret)
	rt.Verify()
}

func (h *tActorHarness) allowance(rt *mock.Runtime, approver, approvee addr.Address) abi.TokenAmount {
	allowanceParams := token.AllowanceParams{
		Owner:   approver,
		Spender: approvee,
	}

	rt.ExpectValidateCallerAny()
	rt.SetCaller(approver, builtin.AccountActorCodeID)
	ret := rt.Call(h.a.Allowance, &allowanceParams)
	rt.Verify()

	balance := ret.(*abi.TokenAmount)
	return *balance
}

func (h *tActorHarness) transferFrom(rt *mock.Runtime, sender, from, to addr.Address, value abi.TokenAmount) {
	transferFromParams := token.TransferFromParams{
		From:  from,
		To:    to,
		Value: value,
	}

	rt.ExpectValidateCallerAny()
	rt.SetCaller(sender, builtin.AccountActorCodeID)
	ret := rt.Call(h.a.TransferFrom, &transferFromParams)
	assert.Nil(h.t, ret)
	rt.Verify()
}
