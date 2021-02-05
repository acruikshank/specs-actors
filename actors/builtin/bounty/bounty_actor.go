package bounty

import (
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
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
	Duration abi.ChainEpoch
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

	st, err := ConstructState(params.PieceCid, params.Token, params.From, params.Value, params.Duration, params.Bounties)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "could not initialize state: %w", err)
	}

	rt.StateCreate(st)
	return nil
}

type ClaimParams struct {
	NewDealID *abi.DealID
}

type payment struct {
	to    addr.Address
	value abi.TokenAmount
}

func (a Actor) Claim(rt runtime.Runtime, params *ClaimParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	// read state to find existing deal ids
	var st State
	rt.StateReadonly(&st)

	var dealIds []abi.DealID
	existingDeals := make(map[abi.DealID]struct{})
	for _, dealBounty := range st.ActiveDeals {
		dealIds = append(dealIds, dealBounty.DealID)
		existingDeals[dealBounty.DealID] = struct{}{}
	}
	if params.NewDealID != nil {
		if _, found := existingDeals[*params.NewDealID]; found {
			rt.Abortf(exitcode.ErrForbidden, "new deal %d is already active", params.NewDealID)
		}

		dealIds = append(dealIds, *params.NewDealID)
	}

	// retrieve deal from market actor (this fails if deal is not active)
	var deals market.GetActiveDealsReturn
	gadParams := &market.GetActiveDealsParams{DealIDs: dealIds}
	code := rt.Send(builtin.StorageMarketActorAddr, builtin.MethodsMarket.GetActiveDeals, gadParams, big.Zero(), &deals)
	builtin.RequireSuccess(rt, code, "failed to retrieve deals")

	var tokenAddr *addr.Address
	var fromAddr addr.Address
	payments := []payment{}

	rt.StateTransaction(&st, func() {
		var nextActiveDeals []DealBounty

		for i, oldDeal := range st.ActiveDeals {
			dealProposal := deals.Proposals[i]
			dealState := deals.States[i]

			var lastActiveEpoch abi.ChainEpoch
			if dealProposal != nil && dealState != nil {
				if dealState.SlashEpoch >= 0 {
					// update next open bounty time if slot is opening up and expiration is less than any other active deal
					lastActiveEpoch = dealState.SlashEpoch
				} else if oldDeal.DealEnd < rt.CurrEpoch() {
					lastActiveEpoch = oldDeal.DealEnd
				} else {
					// deal appears to be active, re-add it
					lastActiveEpoch = rt.CurrEpoch()
					nextActiveDeals = append(nextActiveDeals, DealBounty{
						DealID:   oldDeal.DealID,
						Client:   oldDeal.Client,
						LastPaid: rt.CurrEpoch(),
						DealEnd:  oldDeal.DealEnd,
					})
				}
			} else {
				// assume deal expired
				lastActiveEpoch = oldDeal.DealEnd
			}

			if lastActiveEpoch > oldDeal.LastPaid {
				value := big.Mul(st.Value, big.NewInt(int64(lastActiveEpoch-oldDeal.LastPaid)))
				value = big.Div(value, big.Mul(big.NewInt(int64(st.MaxActiveDeals)), big.NewInt(int64(st.Duration))))

				payments = append(payments, payment{
					to:    oldDeal.Client,
					value: value,
				})
			}
		}

		// now add any new deal if we can
		if params.NewDealID != nil {
			if uint64(len(nextActiveDeals)) >= st.MaxActiveDeals {
				rt.Abortf(exitcode.ErrForbidden, "adding deal %d would active deal limit", params.NewDealID)
			}

			dealProposal := deals.Proposals[len(st.ActiveDeals)]
			dealState := deals.States[len(st.ActiveDeals)]

			if !dealProposal.PieceCID.Equals(st.PieceCid) {
				rt.Abortf(exitcode.ErrNotFound, "proposed bounty deal %d is for wrong piece cid", dealProposal.PieceCID)
			}

			if dealProposal == nil {
				rt.Abortf(exitcode.ErrNotFound, "proposed bounty deal %d not found", params.NewDealID)
			}

			if dealState == nil {
				rt.Abortf(exitcode.ErrNotFound, "proposed bounty deal %d not active", params.NewDealID)
			}

			lastActive := rt.CurrEpoch()
			if dealProposal.EndEpoch < lastActive {
				rt.Abortf(exitcode.ErrNotFound, "proposed bounty deal %d not currently active", params.NewDealID)
			}
			if dealState.SlashEpoch >= 0 && dealState.SlashEpoch < lastActive {
				rt.Abortf(exitcode.ErrNotFound, "proposed bounty deal %d not currently active", params.NewDealID)
			}

			// looks ok, add new deal
			nextActiveDeals = append(nextActiveDeals, DealBounty{
				DealID:   *params.NewDealID,
				Client:   dealProposal.Client,
				LastPaid: rt.CurrEpoch(),
				DealEnd:  dealProposal.EndEpoch,
			})
		}

		st.ActiveDeals = nextActiveDeals

		tokenAddr = st.Token
		fromAddr = st.From
	})

	// getting this far means claim is validated. send payment.
	for _, payment := range payments {
		if tokenAddr != nil {
			transferParams := token.TransferFromParams{
				From:  fromAddr,
				To:    payment.to,
				Value: payment.value,
			}
			code := rt.Send(*tokenAddr, builtin.MethodsToken.TransferFrom, &transferParams, big.Zero(), &builtin.Discard{})
			builtin.RequireSuccess(rt, code, "failed to transfer token %v from %v to %v", tokenAddr, fromAddr, payment.to)
		} else {
			code := rt.Send(payment.to, builtin.MethodSend, nil, payment.value, &builtin.Discard{})
			builtin.RequireSuccess(rt, code, "failed to transfer %v FIL to %v", payment.value, payment.to)
		}
	}

	return nil
}
