package test_test

import (
	"bytes"
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/ipfs/go-cid"
	"testing"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/specs-actors/v3/actors/builtin"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/bounty"
	init_ "github.com/filecoin-project/specs-actors/v3/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/v3/actors/builtin/token"
	"github.com/filecoin-project/specs-actors/v3/support/ipld"
	tutil "github.com/filecoin-project/specs-actors/v3/support/testing"
	vm "github.com/filecoin-project/specs-actors/v3/support/vm"
)

func TestCreateAndClaimTokenBounty(t *testing.T) {
	// create vm
	ctx := context.Background()
	v := vm.NewVMWithSingletons(ctx, t, ipld.NewBlockStoreInMemory())
	v, err := v.WithNetworkVersion(network.Version8)
	require.NoError(t, err)

	// create accounts
	addrs := vm.CreateAccounts(ctx, t, v, 6, big.Mul(big.NewInt(10_000), big.NewInt(1e18)), 93837778)
	founder, worker, client1, client2, client3, client4 := addrs[0], addrs[1], addrs[2], addrs[3], addrs[4], addrs[5]

	// create new coin
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

	// define piece for bounty
	pieceCid := tutil.MakeCID("1", &market.PieceCIDPrefix)

	// create data bounty
	bountyConstructorParams := bounty.ConstructorParams{
		PieceCid: pieceCid,
		Token:    &tokenActorAddr,
		From:     founder,
		Value:    abi.NewTokenAmount(2000),
		Bounties: 3,
	}

	paramBuf = new(bytes.Buffer)
	err = bountyConstructorParams.MarshalCBOR(paramBuf)
	require.NoError(t, err)

	initParam = init_.ExecParams{
		CodeCID:           builtin.BountyActorCodeID,
		ConstructorParams: paramBuf.Bytes(),
	}
	ret = vm.ApplyOk(t, v, founder, builtin.InitActorAddr, big.Zero(), builtin.MethodsInit.Exec, &initParam)
	initRet = ret.(*init_.ExecReturn)
	assert.NotNil(t, initRet)
	bountyActorAddr := initRet.IDAddress

	// founder authorizes bountyActor to make transfers
	approveParams := token.ApproveParams{
		Approvee: bountyActorAddr,
		Value:    abi.NewTokenAmount(6000),
	}
	ret = vm.ApplyOk(t, v, founder, tokenActorAddr, big.Zero(), builtin.MethodsToken.Approve, &approveParams)
	assert.Nil(t, ret)

	// create miner
	minerBalance := big.Mul(big.NewInt(1_000), vm.FIL)
	sealProof := abi.RegisteredSealProof_StackedDrg32GiBV1_1
	params := power.CreateMinerParams{
		Owner:               worker,
		Worker:              worker,
		WindowPoStProofType: abi.RegisteredPoStProof_StackedDrgWindow32GiBV1,
		Peer:                abi.PeerID("not really a peer id"),
	}
	ret = vm.ApplyOk(t, v, addrs[0], builtin.StoragePowerActorAddr, minerBalance, builtin.MethodsPower.CreateMiner, &params)
	minerAddrs, ok := ret.(*power.CreateMinerReturn)
	require.True(t, ok)

	// create some deals with the piece cid
	collateral := big.Mul(big.NewInt(3), vm.FIL)
	vm.ApplyOk(t, v, addrs[1], builtin.StorageMarketActorAddr, collateral, builtin.MethodsMarket.AddBalance, &client1)
	vm.ApplyOk(t, v, addrs[1], builtin.StorageMarketActorAddr, collateral, builtin.MethodsMarket.AddBalance, &client2)
	vm.ApplyOk(t, v, addrs[1], builtin.StorageMarketActorAddr, collateral, builtin.MethodsMarket.AddBalance, &client3)
	vm.ApplyOk(t, v, addrs[1], builtin.StorageMarketActorAddr, collateral, builtin.MethodsMarket.AddBalance, &client4)

	collateral = big.Mul(big.NewInt(64), vm.FIL)
	vm.ApplyOk(t, v, worker, builtin.StorageMarketActorAddr, collateral, builtin.MethodsMarket.AddBalance, &minerAddrs.IDAddress)

	// create 4 deals, each with the right piece cid
	dealIDs := []abi.DealID{}
	dealStart := v.GetEpoch() + miner.MaxProveCommitDuration[sealProof]
	deals := publishDeal(t, v, worker, client1, minerAddrs.IDAddress, pieceCid, "deal1", 1<<32, false, dealStart, 200*builtin.EpochsInDay)
	dealIDs = append(dealIDs, deals.IDs...)
	deals = publishDeal(t, v, worker, client2, minerAddrs.IDAddress, pieceCid, "deal2", 1<<32, false, dealStart, 200*builtin.EpochsInDay)
	dealIDs = append(dealIDs, deals.IDs...)
	deals = publishDeal(t, v, worker, client3, minerAddrs.IDAddress, pieceCid, "deal3", 1<<32, false, dealStart, 200*builtin.EpochsInDay)
	dealIDs = append(dealIDs, deals.IDs...)
	deals = publishDeal(t, v, worker, client4, minerAddrs.IDAddress, pieceCid, "deal4", 1<<32, false, dealStart, 200*builtin.EpochsInDay)
	dealIDs = append(dealIDs, deals.IDs...)

	//
	// Precommit, Prove, Verify and PoSt committed capacity sector
	//

	sectorNumber := abi.SectorNumber(101)
	sealedCid := tutil.MakeCID("101", &miner.SealedCIDPrefix)
	preCommitParams := miner.PreCommitSectorParams{
		SealProof:       sealProof,
		SectorNumber:    sectorNumber,
		SealedCID:       sealedCid,
		SealRandEpoch:   v.GetEpoch() - 1,
		DealIDs:         dealIDs,
		Expiration:      v.GetEpoch() + 220*builtin.EpochsInDay,
		ReplaceCapacity: false,
	}
	vm.ApplyOk(t, v, worker, minerAddrs.RobustAddress, big.Zero(), builtin.MethodsMiner.PreCommitSector, &preCommitParams)

	// advance time to min seal duration
	proveTime := v.GetEpoch() + miner.PreCommitChallengeDelay + 1
	v, _ = vm.AdvanceByDeadlineTillEpoch(t, v, minerAddrs.IDAddress, proveTime)

	// Prove commit sector after max seal duration
	v, err = v.WithEpoch(proveTime)
	require.NoError(t, err)
	proveCommitParams := miner.ProveCommitSectorParams{
		SectorNumber: sectorNumber,
	}
	vm.ApplyOk(t, v, worker, minerAddrs.RobustAddress, big.Zero(), builtin.MethodsMiner.ProveCommitSector, &proveCommitParams)

	// In the same epoch, trigger cron to validate prove commit
	vm.ApplyOk(t, v, builtin.SystemActorAddr, builtin.CronActorAddr, big.Zero(), builtin.MethodsCron.EpochTick, nil)

	//
	// Collect Bounties now that sector is sealed and deals are active
	//

	v, err = v.WithEpoch(v.GetEpoch() + 1)
	require.NoError(t, err)

	claimParams := bounty.ClaimParams{
		DealID: dealIDs[0],
	}
	ret = vm.ApplyOk(t, v, client1, bountyActorAddr, big.Zero(), builtin.MethodsBounty.Claim, &claimParams)
	assert.Nil(t, ret)

	// confirm client has bounty
	ret = vm.ApplyOk(t, v, client1, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &client1)
	balance := ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(2000))

	t.Run("invalid deal is forbidden", func(t *testing.T) {
		v2, err := v.WithEpoch(v.GetEpoch()) // clone vm
		require.NoError(t, err)

		claimParams = bounty.ClaimParams{
			DealID: abi.DealID(42),
		}
		_, code := v2.ApplyMessage(client1, bountyActorAddr, big.Zero(), builtin.MethodsBounty.Claim, &claimParams)
		require.NotEqual(t, exitcode.Ok, code)
	})

	t.Run("duplicate claim is forbidden", func(t *testing.T) {
		v2, err := v.WithEpoch(v.GetEpoch()) // clone vm
		require.NoError(t, err)

		claimParams = bounty.ClaimParams{
			DealID: dealIDs[0],
		}
		_, code := v2.ApplyMessage(client1, bountyActorAddr, big.Zero(), builtin.MethodsBounty.Claim, &claimParams)
		require.Equal(t, exitcode.ErrForbidden, code)
	})

	// make second claim and confirm client2 has bounty (claim and confirm from client1)
	claimParams = bounty.ClaimParams{
		DealID: dealIDs[1],
	}
	ret = vm.ApplyOk(t, v, client1, bountyActorAddr, big.Zero(), builtin.MethodsBounty.Claim, &claimParams)
	assert.Nil(t, ret)

	// confirm client2 has bounty
	ret = vm.ApplyOk(t, v, client1, tokenActorAddr, big.Zero(), builtin.MethodsToken.BalanceOf, &client2)
	balance = ret.(*abi.TokenAmount)
	assert.Equal(t, *balance, abi.NewTokenAmount(2000))

	// make third claim for confirm client3
	claimParams = bounty.ClaimParams{
		DealID: dealIDs[2],
	}
	ret = vm.ApplyOk(t, v, client1, bountyActorAddr, big.Zero(), builtin.MethodsBounty.Claim, &claimParams)
	assert.Nil(t, ret)

	t.Run("fourth claim is forbidden", func(t *testing.T) {
		claimParams = bounty.ClaimParams{
			DealID: dealIDs[3],
		}
		_, code := v.ApplyMessage(client1, bountyActorAddr, big.Zero(), builtin.MethodsBounty.Claim, &claimParams)
		require.Equal(t, exitcode.ErrForbidden, code)
	})
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

func publishDeal(t *testing.T, v *vm.VM, provider, dealClient, minerID address.Address, pieceCid cid.Cid, dealLabel string,
	pieceSize abi.PaddedPieceSize, verifiedDeal bool, dealStart abi.ChainEpoch, dealLifetime abi.ChainEpoch,
) *market.PublishStorageDealsReturn {
	deal := market.DealProposal{
		PieceCID:             pieceCid,
		PieceSize:            pieceSize,
		VerifiedDeal:         verifiedDeal,
		Client:               dealClient,
		Provider:             minerID,
		Label:                dealLabel,
		StartEpoch:           dealStart,
		EndEpoch:             dealStart + dealLifetime,
		StoragePricePerEpoch: abi.NewTokenAmount(1 << 20),
		ProviderCollateral:   big.Mul(big.NewInt(2), vm.FIL),
		ClientCollateral:     big.Mul(big.NewInt(1), vm.FIL),
	}

	publishDealParams := market.PublishStorageDealsParams{
		Deals: []market.ClientDealProposal{{
			Proposal:        deal,
			ClientSignature: crypto.Signature{},
		}},
	}
	ret, code := v.ApplyMessage(provider, builtin.StorageMarketActorAddr, big.Zero(), builtin.MethodsMarket.PublishStorageDeals, &publishDealParams)
	require.Equal(t, exitcode.Ok, code)

	expectedPublishSubinvocations := []vm.ExpectInvocation{
		{To: minerID, Method: builtin.MethodsMiner.ControlAddresses, SubInvocations: []vm.ExpectInvocation{}},
		{To: builtin.RewardActorAddr, Method: builtin.MethodsReward.ThisEpochReward, SubInvocations: []vm.ExpectInvocation{}},
		{To: builtin.StoragePowerActorAddr, Method: builtin.MethodsPower.CurrentTotalPower, SubInvocations: []vm.ExpectInvocation{}},
	}

	if verifiedDeal {
		expectedPublishSubinvocations = append(expectedPublishSubinvocations, vm.ExpectInvocation{
			To:             builtin.VerifiedRegistryActorAddr,
			Method:         builtin.MethodsVerifiedRegistry.UseBytes,
			SubInvocations: []vm.ExpectInvocation{},
		})
	}

	vm.ExpectInvocation{
		To:             builtin.StorageMarketActorAddr,
		Method:         builtin.MethodsMarket.PublishStorageDeals,
		SubInvocations: expectedPublishSubinvocations,
	}.Matches(t, v.LastInvocation())

	return ret.(*market.PublishStorageDealsReturn)
}
