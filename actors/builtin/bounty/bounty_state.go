package bounty

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	cid "github.com/ipfs/go-cid"
)

type State struct {
	// Cid for which this actor will pay the bounty
	PieceCid cid.Cid

	// If present, bounty will be paid in this token
	Token *addr.Address

	// If Token present, bounty will be paid from this account (bounty actor must be approved).
	From addr.Address

	// Total amount to pay for storage time
	Value abi.TokenAmount

	// Duration amount of time covered by bounty
	Duration abi.ChainEpoch

	// Number of remaining bounties to pay
	MaxActiveDeals uint64

	// stores active deal ids and last epoch for which they were paid
	ActiveDeals []DealBounty
}

type DealBounty struct {
	DealID   abi.DealID
	Client   addr.Address
	LastPaid abi.ChainEpoch
	Start    abi.ChainEpoch
	DealEnd  abi.ChainEpoch
}

func ConstructState(pieceCid cid.Cid, token *addr.Address, from addr.Address, value abi.TokenAmount, duration abi.ChainEpoch, bounties uint64) (*State, error) {
	return &State{
		PieceCid:       pieceCid,
		Token:          token,
		From:           from,
		Value:          value,
		Duration:       duration,
		MaxActiveDeals: bounties,
		ActiveDeals:    []DealBounty{},
	}, nil
}
