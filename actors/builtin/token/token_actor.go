package token

import (
	"github.com/ipfs/go-cid"

	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/exitcode"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/runtime"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
)

type Actor struct{}

func (a Actor) Exports() []interface{} {
	return []interface{}{
		builtin.MethodConstructor: a.Constructor,
		2:                         a.Name,
		3:                         a.Symbol,
		4:                         a.Decimals,
		5:                         a.TotalSupply,
		6:                         a.BalanceOf,
		7:                         a.Transfer,
		8:                         a.Approve,
		9:                         a.Allowance,
		10:                        a.TransferFrom,
		11:                        a.Icon,
	}
}

func (a Actor) Code() cid.Cid {
	return builtin.TokenActorCodeID
}

func (a Actor) State() cbor.Er {
	return new(State)
}

var _ runtime.VMActor = Actor{}

type ConstructorParams struct {
	Name          string
	Symbol        string
	Icon          []byte
	Decimals      uint64
	TotalSupply   abi.TokenAmount
	SystemAccount addr.Address
}

func (a Actor) Constructor(rt runtime.Runtime, params *ConstructorParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerIs(builtin.InitActorAddr)

	// Ensure token starts with funds
	if params.TotalSupply.LessThanEqual(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "must have initial total supply greater than zero")
	}

	// Ensure system account exists
	resolvedSystem, err := builtin.ResolveToIDAddr(rt, params.SystemAccount)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve system address %v: %w", params.SystemAccount, err)
	}

	// Ensure system account is signer
	codeCID, ok := rt.GetActorCodeCID(resolvedSystem)
	if !ok {
		rt.Abortf(exitcode.ErrIllegalArgument, "no code for system address %v", resolvedSystem)
	}
	if !builtin.IsPrincipal(codeCID) {
		rt.Abortf(exitcode.ErrForbidden, "actor %v must be a principal account (%v), was %v", params.SystemAccount,
			builtin.AccountActorCodeID, codeCID)
	}

	st, err := ConstructState(adt.AsStore(rt), params.Name, params.Symbol, params.Icon, params.Decimals, params.TotalSupply, resolvedSystem)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "could not initialize state: %w", err)
	}

	rt.StateCreate(st)
	return nil
}

// Get name of token
func (a Actor) Name(rt runtime.Runtime, value abi.EmptyValue) string {
	rt.ValidateImmediateCallerAcceptAny()

	var st State
	rt.StateReadonly(&st)
	return st.Name
}

// Get symbol of token
func (a Actor) Symbol(rt runtime.Runtime, value abi.EmptyValue) string {
	rt.ValidateImmediateCallerAcceptAny()

	var st State
	rt.StateReadonly(&st)
	return st.Symbol
}

// Get symbol of token
func (a Actor) Icon(rt runtime.Runtime, value abi.EmptyValue) []byte {
	rt.ValidateImmediateCallerAcceptAny()

	var st State
	rt.StateReadonly(&st)
	return st.Icon
}

// Get decimals used by token
func (a Actor) Decimals(rt runtime.Runtime, value abi.EmptyValue) uint64 {
	rt.ValidateImmediateCallerAcceptAny()

	var st State
	rt.StateReadonly(&st)
	return st.Decimals
}

// Get total supply of token
func (a Actor) TotalSupply(rt runtime.Runtime, value abi.EmptyValue) abi.TokenAmount {
	rt.ValidateImmediateCallerAcceptAny()

	var st State
	rt.StateReadonly(&st)
	return st.TotalSupply
}

// Get balance for address
func (a Actor) BalanceOf(rt runtime.Runtime, account addr.Address) abi.TokenAmount {
	rt.ValidateImmediateCallerAcceptAny()

	// resolve address
	resolvedAccount, err := builtin.ResolveToIDAddr(rt, account)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve account address %v: %w", account, err)
	}

	var st State
	rt.StateReadonly(&st)

	balance, err := st.BalanceOf(adt.AsStore(rt), resolvedAccount)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, "failed to retrieve balance: %w", err)
	}

	return balance
}

type TransferParams struct {
	To    addr.Address
	Value abi.TokenAmount
}

// Transfer balance to another account
func (a Actor) Transfer(rt runtime.Runtime, params *TransferParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny() // only addresses with balances will work

	// Value must be positive
	if params.Value.LessThanEqual(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "transfer value must be positive")
	}

	// resolve to address
	resolvedTo, err := builtin.ResolveToIDAddr(rt, params.To)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve account address %v: %w", params.To, err)
	}

	var st State
	rt.StateTransaction(&st, func() {
		insufficientFunds, err := st.Transfer(adt.AsStore(rt), rt.Caller(), resolvedTo, params.Value)
		if err != nil {
			if insufficientFunds {
				rt.Abortf(exitcode.ErrInsufficientFunds, err.Error())
			}
			rt.Abortf(exitcode.ErrIllegalState, "failed to transfer funds: %w", err)
		}
	})

	return nil
}

type ApproveParams struct {
	Approvee addr.Address
	Value    abi.TokenAmount
}

// Approve another address to transfer on this account's behalf
func (a Actor) Approve(rt runtime.Runtime, params *ApproveParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny()

	// Value must be positive
	if params.Value.LessThanEqual(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "transfer value must be positive")
	}

	// resolve approvee address
	resolvedApprovee, err := builtin.ResolveToIDAddr(rt, params.Approvee)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve approvee address %v: %w", params.Approvee, err)
	}

	var st State
	rt.StateTransaction(&st, func() {
		err := st.Approve(adt.AsStore(rt), rt.Caller(), resolvedApprovee, params.Value)
		if err != nil {
			rt.Abortf(exitcode.ErrIllegalState, "failed to approve %w: %w", resolvedApprovee, err)
		}
	})

	return nil
}

type AllowanceParams struct {
	Owner   addr.Address
	Spender addr.Address
}

// retrieve how much another address is authorized to spend
func (a Actor) Allowance(rt runtime.Runtime, params *AllowanceParams) abi.TokenAmount {
	rt.ValidateImmediateCallerAcceptAny()

	// resolve owner address
	resolvedOwner, err := builtin.ResolveToIDAddr(rt, params.Owner)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve owner address %v: %w", params.Owner, err)
	}

	// resolve spender address
	resolvedSpender, err := builtin.ResolveToIDAddr(rt, params.Spender)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve spender address %v: %w", params.Spender, err)
	}

	var st State
	rt.StateReadonly(&st)

	amount, err := st.Allowance(adt.AsStore(rt), resolvedOwner, resolvedSpender)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalState, err.Error())
	}

	return amount
}

type TransferFromParams struct {
	From  addr.Address
	To    addr.Address
	Value abi.TokenAmount
}

func (a Actor) TransferFrom(rt runtime.Runtime, params *TransferFromParams) *abi.EmptyValue {
	rt.ValidateImmediateCallerAcceptAny() // only addresses with balances will work

	// Value must be positive
	if params.Value.LessThanEqual(big.Zero()) {
		rt.Abortf(exitcode.ErrIllegalArgument, "transfer value must be positive")
	}

	// resolve from address
	resolvedFrom, err := builtin.ResolveToIDAddr(rt, params.From)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve from account address %v: %w", params.From, err)
	}

	// resolve to address
	resolvedTo, err := builtin.ResolveToIDAddr(rt, params.To)
	if err != nil {
		rt.Abortf(exitcode.ErrIllegalArgument, "failed to resolve to account address %v: %w", params.To, err)
	}

	var st State
	rt.StateTransaction(&st, func() {
		// confirm sender is allowed to transfer this much (and deduct that value from allowance)
		insufficientFunds, err := st.DeductAllowance(adt.AsStore(rt), resolvedFrom, rt.Caller(), params.Value)
		if err != nil {
			if insufficientFunds {
				rt.Abortf(exitcode.ErrForbidden, "%v has insufficient allowance to transfer %v for %v",
					rt.Caller(), params.Value, params.From)
			}
		}

		insufficientFunds, err = st.Transfer(adt.AsStore(rt), resolvedFrom, resolvedTo, params.Value)
		if err != nil {
			if insufficientFunds {
				rt.Abortf(exitcode.ErrInsufficientFunds, err.Error())
			}
			rt.Abortf(exitcode.ErrIllegalState, "failed to transfer funds: %w", err)
		}
	})

	return nil
}
