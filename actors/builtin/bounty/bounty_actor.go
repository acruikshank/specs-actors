package bounty

import (
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	"github.com/ipfs/go-cid"

	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/runtime"
)

type Actor struct{}

func (a Actor) Exports() []interface{} {
	return []interface{}{
		builtin.MethodConstructor: a.Constructor,
		2:                         a.Claim,
	}
}

func (a Actor) Code() cid.Cid {
	return builtin.BountyActorCodeID
}

func (a Actor) State() cbor.Er {
	return new(State)
}

var _ runtime.VMActor = Actor{}

type ConstructorParams struct {
	PieceCid cid.Cid
	Token    *addr.Address
	From     addr.Address
	Value    abi.TokenAmount
	Bounties uint64
}

func (a Actor) Constructor(rt runtime.Runtime, params *ConstructorParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerIs(builtin.InitActorAddr)

	// Ensure amount greater than zero
	if params.Value.LessThanEqual(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "must have value greater than zero")
	}

	// If no token, ensure balance is sufficient to cover all bounties
	totalValue := big.Mul(params.Value, big.NewInt(int64(params.Bounties)))
	if params.Token == nil && rt.CurrentBalance().LessThan(totalValue) {
		rt.Abortf(exitcode.ErrIllegalArgument, "bounty actor balance must be cover total value of bounties")
	}

	st, err := ConstructState(adt.AsStore(rt), params.PieceCid, params.Token, params.From, params.Value, params.Bounties)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "could not initialize state: %w", err)
	}

	rt.StateCreate(st)
	return nil
}

type ClaimParams struct {
	DealID abi.DealID
}

func (a Actor) Claim(rt runtime.Runtime, params *ClaimParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	// retrieve deal from market actor (this fails if deal is not active)
	var proposal market.DealProposal
	gadParams := &market.GetActiveDealParams{DealID: params.DealID}
	code := rt.Send(builtin.StorageMarketActorAddr, builtin.MethodsMarket.GetActiveDeal, gadParams, big.Zero(), &proposal)
	builtin.RequireSuccess(rt, code, "failed to retrieve proposal")

	var tokenAddr *addr.Address
	var fromAddr addr.Address
	var value abi.TokenAmount

	var st State
	rt.StateTransaction(&st, func() {
		if st.Bounties < 1 {
			rt.Abortf(exitcode.ErrForbidden, "all bounties have been claimed")
		}

		if !proposal.PieceCID.Equals(st.PieceCid) {
			rt.Abortf(exitcode.ErrIllegalArgument, "deal piece cid %v does not match bounty piece cid %v", proposal.PieceCID, st.PieceCid)
		}

		paid, err := adt.AsMap(adt.AsStore(rt), st.Paid, builtin.DefaultHamtBitwidth)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "could not open paid map")

		dealKey := abi.IntKey(int64(params.DealID))
		var amount abi.TokenAmount
		found, err := paid.Get(dealKey, &amount)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "could not look up deal %v", params.DealID)

		if found {
			rt.Abortf(exitcode.ErrForbidden, "deal %v already claimed", params.DealID)
		}

		err = paid.Put(dealKey, &st.Value)
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "could not store deal for claim")

		st.Paid, err = paid.Root()
		builtin.RequireNoErr(rt, err, exitcode.ErrIllegalState, "could not retrieve paid root")

		st.Bounties -= 1
		tokenAddr = st.Token
		value = st.Value
		fromAddr = st.From
	})

	// getting this far means claim is validated. send payment.
	if tokenAddr != nil {
		transferParams := token.TransferFromParams{
			From:  fromAddr,
			To:    proposal.Client,
			Value: value,
		}
		code := rt.Send(*tokenAddr, builtin.MethodsToken.TransferFrom, &transferParams, big.Zero(), &builtin.Discard{})
		builtin.RequireSuccess(rt, code, "failed to transfer token %v from %v to %v", tokenAddr, fromAddr, proposal.Client)
	} else {
		code := rt.Send(proposal.Client, builtin.MethodSend, nil, value, &builtin.Discard{})
		builtin.RequireSuccess(rt, code, "failed to transfer %v FIL to %v", value, proposal.Client)
	}

	return nil
}
