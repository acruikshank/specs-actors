package token

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
)

type State struct {
	// Name of token
	Name string

	// Symbol for token
	Symbol string

	// Image for icon
	Icon []byte

	// Number decimals represented by 1 token unit
	Decimals uint64

	// Total amount of tokens in this actor
	TotalSupply abi.TokenAmount

	// Balance sheet for all token owners
	Balances cid.Cid // Map, HAMT[address]TokenAmount

	// Approvals to transfer from another account
	Approvals cid.Cid // MultiMap, HAMT[address]HAMT[address]TokenAmount
}

func ConstructState(store adt.Store, name string, symbol string, icon []byte, decimals uint64, supply abi.TokenAmount, owner addr.Address) (*State, error) {
	// create empty map for balances
	balances, err := adt.MakeEmptyMap(store, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("failed to create balances map: %w", err)
	}

	// store all initial token in owner's account
	err = balances.Put(abi.AddrKey(owner), &supply)
	if err != nil {
		return nil, xerrors.Errorf("failed to initial balance of %v for %v: %w", supply, owner, err)
	}

	// get new balances cid
	balanceRoot, err := balances.Root()
	if err != nil {
		return nil, xerrors.Errorf("failed to get root of balance table: %w", err)
	}

	// create empty multimap for approvals
	approvalsRoot, err := adt.StoreEmptyMap(store, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("failed to create approvals map: %w", err)
	}

	return &State{
		Name:        name,
		Symbol:      symbol,
		Icon:        icon,
		Decimals:    decimals,
		TotalSupply: supply,
		Balances:    balanceRoot,
		Approvals:   approvalsRoot,
	}, nil
}

func (st *State) BalanceOf(store adt.Store, account addr.Address) (abi.TokenAmount, error) {
	balances, err := adt.AsMap(store, st.Balances, builtin.DefaultHamtBitwidth)
	if err != nil {
		return big.Zero(), xerrors.Errorf("failed to load balances: %w", err)
	}

	var balance abi.TokenAmount
	found, err := balances.Get(abi.AddrKey(account), &balance)
	if err != nil {
		return big.Zero(), xerrors.Errorf("failed to load balance for %v: %w", account, err)
	}

	if !found {
		return big.Zero(), nil
	}

	return balance, nil
}

func (st *State) Transfer(store adt.Store, from addr.Address, to addr.Address, value abi.TokenAmount) (insufficientFunds bool, err error) {
	balances, err := adt.AsMap(store, st.Balances, builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, xerrors.Errorf("failed to load balances: %w", err)
	}

	var fromBalance abi.TokenAmount
	found, err := balances.Get(abi.AddrKey(from), &fromBalance)
	if err != nil {
		return false, xerrors.Errorf("failed to load balance for %v: %w", from, err)
	}

	if !found {
		fromBalance = big.Zero()
	}

	if fromBalance.LessThan(value) {
		return true, xerrors.Errorf("%v has insufficent funds (%v) to transfer %v to %", from, fromBalance, value, to)
	}

	var toBalance abi.TokenAmount
	found, err = balances.Get(abi.AddrKey(to), &toBalance)
	if err != nil {
		return false, xerrors.Errorf("failed to load balance for %v: %w", to, err)
	}

	if !found {
		toBalance = big.Zero()
	}

	newFromBalance := big.Add(fromBalance, value.Neg())
	newToBalance := big.Add(toBalance, value)

	err = balances.Put(abi.AddrKey(from), &newFromBalance)
	if err != nil {
		return false, xerrors.Errorf("failed to store balance: %w", err)
	}

	err = balances.Put(abi.AddrKey(to), &newToBalance)
	if err != nil {
		return false, xerrors.Errorf("failed to store balance: %w", err)
	}

	balanceRoot, err := balances.Root()
	if err != nil {
		return false, xerrors.Errorf("failed to store balance: %w", err)
	}
	st.Balances = balanceRoot

	return false, nil
}

func (st *State) Approve(store adt.Store, approver addr.Address, approvee addr.Address, value abi.TokenAmount) error {
	approvals, err := adt.AsMap(store, st.Approvals, builtin.DefaultHamtBitwidth)
	if err != nil {
		return xerrors.Errorf("could not open approvals table: %w", err)
	}

	var approveesCid cbg.CborCid
	found, err := approvals.Get(abi.AddrKey(approver), &approveesCid)
	if err != nil {
		return xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
	}

	var approvees *adt.Map
	if found {
		approvees, err = adt.AsMap(store, cid.Cid(approveesCid), builtin.DefaultHamtBitwidth)
		if err != nil {
			return xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
		}
	} else {
		approvees, err = adt.MakeEmptyMap(store, builtin.DefaultHamtBitwidth)
		if err != nil {
			return xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
		}
	}

	var approveeBalance abi.TokenAmount
	found, err = approvees.Get(abi.AddrKey(approvee), &approveeBalance)
	if err != nil {
		return xerrors.Errorf("could not get balance for approvee %v: %w", approvee, err)
	}

	if !found {
		approveeBalance = big.Zero()
	}

	approveeNewBalance := big.Add(approveeBalance, value)

	err = approvees.Put(abi.AddrKey(approvee), &approveeNewBalance)
	if err != nil {
		return xerrors.Errorf("could not get save balance for approvee %v: %w", approvee, err)
	}

	approveeRoot, err := approvees.Root()
	if err != nil {
		return xerrors.Errorf("could not get approvees root: %w", err)
	}

	cbgRoot := cbg.CborCid(approveeRoot)
	err = approvals.Put(abi.AddrKey(approver), &cbgRoot)
	if err != nil {
		return xerrors.Errorf("could not get save approvals: %w", err)
	}

	approvalsRoot, err := approvals.Root()
	if err != nil {
		return xerrors.Errorf("could not get approvals root: %w", err)
	}

	st.Approvals = approvalsRoot

	return nil
}

func (st *State) Allowance(store adt.Store, approver addr.Address, approvee addr.Address) (abi.TokenAmount, error) {
	approvals, err := adt.AsMap(store, st.Approvals, builtin.DefaultHamtBitwidth)
	if err != nil {
		return big.Zero(), xerrors.Errorf("could not open approvals table: %w", err)
	}

	var approveesCid cbg.CborCid
	found, err := approvals.Get(abi.AddrKey(approver), &approveesCid)
	if err != nil {
		return big.Zero(), xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
	}

	if !found {
		return big.Zero(), nil
	}

	approvees, err := adt.AsMap(store, cid.Cid(approveesCid), builtin.DefaultHamtBitwidth)
	if err != nil {
		return big.Zero(), xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
	}

	var approveeBalance abi.TokenAmount
	found, err = approvees.Get(abi.AddrKey(approvee), &approveeBalance)
	if err != nil {
		return big.Zero(), xerrors.Errorf("could not get balance for approvee %v: %w", approvee, err)
	}

	if !found {
		return big.Zero(), nil
	}

	return approveeBalance, nil
}

func (st *State) DeductAllowance(store adt.Store, approver addr.Address, approvee addr.Address, value abi.TokenAmount) (insufficientAllowance bool, err error) {
	approvals, err := adt.AsMap(store, st.Approvals, builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, xerrors.Errorf("could not open approvals table: %w", err)
	}

	var approveesCid cbg.CborCid
	found, err := approvals.Get(abi.AddrKey(approver), &approveesCid)
	if err != nil {
		return false, xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
	}

	if !found {
		return true, xerrors.Errorf("insufficient approval budget (0) for %v to transfer from %v: %w",
			approvee, approver, err)
	}

	approvees, err := adt.AsMap(store, cid.Cid(approveesCid), builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, xerrors.Errorf("could not open approvees table for %v: %w", approver, err)
	}

	var approveeBalance abi.TokenAmount
	found, err = approvees.Get(abi.AddrKey(approvee), &approveeBalance)
	if err != nil {
		return false, xerrors.Errorf("could not get balance for approvee %v: %w", approvee, err)
	}

	if !found {
		return true, xerrors.Errorf("insufficient approval budget (0) for %v to transfer from %v: %w",
			approvee, approver, err)
	}

	if approveeBalance.LessThan(value) {
		return true, xerrors.Errorf("insufficient approval budget (%v) for %v to transfer from %v: %w",
			approveeBalance, approvee, approver, err)
	}

	newApproveeBalance := big.Add(approveeBalance, value.Neg())

	err = approvees.Put(abi.AddrKey(approvee), &newApproveeBalance)
	if err != nil {
		return false, xerrors.Errorf("could not get save balance for approvee %v: %w", approvee, err)
	}

	approveeRoot, err := approvees.Root()
	if err != nil {
		return false, xerrors.Errorf("could not get approvees root: %w", err)
	}

	cbgRoot := cbg.CborCid(approveeRoot)
	err = approvals.Put(abi.AddrKey(approver), &cbgRoot)
	if err != nil {
		return false, xerrors.Errorf("could not get save approvals: %w", err)
	}

	approvalsRoot, err := approvals.Root()
	if err != nil {
		return false, xerrors.Errorf("could not get approvals root: %w", err)
	}

	st.Approvals = approvalsRoot

	return false, nil
}
