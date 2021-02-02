package token_test

import (
	"testing"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/support/mock"
)

func TestExports(t *testing.T) {
	mock.CheckActorExports(t, token.Actor{})
}

type tActorHarness struct {
	a token.Actor
	t testing.TB
}

//func (h *tActorHarness) constructAndVerify(rt *mock.Runtime, name string, symbol string, icon []byte, decimals uint64, supply abi.TokenAmount) {
//	constructParams := multisig.ConstructorParams{
//		Signers:               signers,
//		NumApprovalsThreshold: numApprovalsThresh,
//		UnlockDuration:        unlockDuration,
//		StartEpoch:            startEpoch,
//	}
//
//	rt.ExpectValidateCallerAddr(builtin.InitActorAddr)
//	ret := rt.Call(h.a.Constructor, &constructParams)
//	assert.Nil(h.t, ret)
//	rt.Verify()
//}
