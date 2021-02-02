package token

import (
	addr "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	cid "github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/util/adt"
)

type State struct {
	// Name of token
	Name string

	// Symbol for token
	Symbol string

	// Number decimals represented by 1 token unit
	Decimals uint64

	// Total amount of tokens in this actor
	TotalSupply abi.TokenAmount

	// Balance sheet for all token owners
	Balances cid.Cid // Map, HAMT[address]TokenAmount
}

func ConstructState(store adt.Store, name string, symbol string, decimals uint64, supply abi.TokenAmount, owner addr.Address) (*State, error) {
	// create empty map for balances
	balances, err := adt.MakeEmptyMap(store, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("failed to create empty map: %w", err)
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

	return &State{
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		TotalSupply: supply,
		Balances:    balanceRoot,
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
