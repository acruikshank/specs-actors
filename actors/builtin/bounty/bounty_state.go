package bounty

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
	cid "github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

type State struct {
	// Cid for which this actor will pay the bounty
	PieceCid cid.Cid

	// If present, bounty will be paid in this token
	Token *addr.Address

	// If Token present, bounty will be paid from this account (bounty actor must be approved).
	From addr.Address

	// Amount to pay for each token
	Value abi.TokenAmount

	// Number of remaining bounties to pay
	Bounties uint64

	// store which bounties have already been paid
	Paid cid.Cid // Map, HAMT[DealID]TokenAmount
}

func ConstructState(store adt.Store, pieceCid cid.Cid, token *addr.Address, from addr.Address, value abi.TokenAmount, bounties uint64) (*State, error) {
	// create empty map for paid
	paidCid, err := adt.StoreEmptyMap(store, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("failed to create paid map: %w", err)
	}

	return &State{
		PieceCid: pieceCid,
		Token:    token,
		From:     from,
		Value:    value,
		Bounties: bounties,
		Paid:     paidCid,
	}, nil
}
