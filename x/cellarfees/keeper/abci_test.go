package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/mock/gomock"
	gravitytypes "github.com/peggyjv/gravity-bridge/module/v4/x/gravity/types"
	appParams "github.com/peggyjv/sommelier/v7/app/params"
	auctionTypes "github.com/peggyjv/sommelier/v7/x/auction/types"
	cellarfeesTypes "github.com/peggyjv/sommelier/v7/x/cellarfees/types"
)

func (suite *KeeperTestSuite) TestBeginBlockerZeroRewardsBalance() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())

	require := suite.Require()

	params := cellarfeesTypes.DefaultParams()
	cellarfeesKeeper.SetParams(ctx, params)

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(appParams.BaseCoinUnit, sdk.ZeroInt()))
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// Note EndBlocker is only run once for completeness, since it has no code in it
	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })
	require.NotPanics(func() { cellarfeesKeeper.EndBlocker(ctx) })
}

func (suite *KeeperTestSuite) TestBeginBlockerWithRewardBalanceAndPreviousPeakZero() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())

	require := suite.Require()

	params := cellarfeesTypes.DefaultParams()
	cellarfeesKeeper.SetParams(ctx, params)

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	rewardSupply := sdk.NewCoin(appParams.BaseCoinUnit, sdk.NewInt(1000000))
	emissionPeriod := sdk.NewInt(int64(params.RewardEmissionPeriod))
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(appParams.BaseCoinUnit, rewardSupply.Amount))
	suite.cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, sdk.NewInt(0))

	expectedEmissionAmount := rewardSupply.Amount.Quo(emissionPeriod)
	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, expectedEmissionAmount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })
}

func (suite *KeeperTestSuite) TestBeginBlockerWithRewardBalanceAndHigherPreviousPeak() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())
	require := suite.Require()

	params := cellarfeesTypes.DefaultParams()
	cellarfeesKeeper.SetParams(ctx, params)

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	rewardSupply := sdk.NewCoin(appParams.BaseCoinUnit, sdk.NewInt(1000000))
	emissionPeriod := sdk.NewInt(int64(params.RewardEmissionPeriod))
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(appParams.BaseCoinUnit, rewardSupply.Amount))

	previousPeakAmount := sdk.NewInt(2000000)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, previousPeakAmount)

	expectedEmissionAmount := previousPeakAmount.Quo(emissionPeriod)
	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, expectedEmissionAmount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })
}

func (suite *KeeperTestSuite) TestBeginBlockerWithRewardBalanceAndLowerPreviousPeak() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())

	require := suite.Require()

	params := cellarfeesTypes.DefaultParams()
	cellarfeesKeeper.SetParams(ctx, params)

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	rewardSupply := sdk.NewCoin(appParams.BaseCoinUnit, sdk.NewInt(1000000))
	emissionPeriod := sdk.NewInt(int64(params.RewardEmissionPeriod))
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(appParams.BaseCoinUnit, rewardSupply.Amount))

	previousPeakAmount := sdk.NewInt(500000)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, previousPeakAmount)

	expectedEmissionAmount := rewardSupply.Amount.Quo(emissionPeriod)
	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, expectedEmissionAmount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })
}

// If the emission calculation underflows to zero, it should be set to 1
func (suite *KeeperTestSuite) TestBeginBlockerEmissionCalculationUnderflowsToZero() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())
	require := suite.Require()

	params := cellarfeesTypes.DefaultParams()
	cellarfeesKeeper.SetParams(ctx, params)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, sdk.ZeroInt())

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	rewardSupply := sdk.NewCoin(appParams.BaseCoinUnit, sdk.NewInt(1))
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(appParams.BaseCoinUnit, rewardSupply.Amount))

	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, sdk.OneInt())
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })
}

// If the calculated emission is greater than the remaining supply, it should be set to the remaining supply
func (suite *KeeperTestSuite) TestBeginBlockerEmissionGreaterThanRewardSupply() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())

	require := suite.Require()
	params := cellarfeesTypes.DefaultParams()
	cellarfeesKeeper.SetParams(ctx, params)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, sdk.NewInt(1000000))

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	rewardSupply := sdk.NewCoin(appParams.BaseCoinUnit, sdk.NewInt(1))
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(appParams.BaseCoinUnit, rewardSupply.Amount))

	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, rewardSupply.Amount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })
}

func (suite *KeeperTestSuite) TestAuctionBeginWithSufficientFunds() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())
	suite.SetupHooksTests(ctx, cellarfeesKeeper)

	require := suite.Require()
	params := cellarfeesTypes.DefaultParams()
	params.AuctionInterval = 1
	cellarfeesKeeper.SetParams(ctx, params)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, sdk.NewInt(1000000))

	hooks := Hooks{k: cellarfeesKeeper}
	event := gravitytypes.SendToCosmosEvent{
		CosmosReceiver: feesAccount.GetAddress().String(),
		Amount:         sdk.NewInt(2),
		EthereumSender: "0x0000000000000000000000000000000000000000",
		TokenContract:  "0x1111111111111111111111111111111111111111",
	}
	cellarID := common.HexToAddress(event.EthereumSender)
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.FeeAccrualCounters{
		Counters: []cellarfeesTypes.FeeAccrualCounter{
			{
				Denom: gravityFeeDenom,
				Count: 1,
			},
		},
	})
	expectedCounters := cellarfeesTypes.FeeAccrualCounters{
		Counters: []cellarfeesTypes.FeeAccrualCounter{
			{
				Denom: gravityFeeDenom,
				Count: 0,
			},
		},
	}

	suite.corkKeeper.EXPECT().HasCellarID(ctx, cellarID).Return(true)
	suite.gravityKeeper.EXPECT().ERC20ToDenomLookup(ctx, common.HexToAddress(event.TokenContract)).Return(false, gravityFeeDenom)
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	require.NotPanics(func() { hooks.AfterSendToCosmosEvent(ctx, event) })

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(gravityFeeDenom, sdk.NewInt(0)))

	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), gravityFeeDenom).Return(sdk.NewCoin(gravityFeeDenom, event.Amount))

	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, event.Amount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)
	suite.auctionKeeper.EXPECT().GetActiveAuctions(ctx).Return([]*auctionTypes.Auction{})
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), gravityFeeDenom).Return(sdk.NewCoin(gravityFeeDenom, event.Amount))
	suite.auctionKeeper.EXPECT().BeginAuction(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })

	require.Equal(expectedCounters, cellarfeesKeeper.GetFeeAccrualCounters(ctx))

}

func (suite *KeeperTestSuite) TestAuctionBeginWithInSufficientFunds() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())
	suite.SetupHooksTests(ctx, cellarfeesKeeper)

	require := suite.Require()
	params := cellarfeesTypes.DefaultParams()
	params.AuctionInterval = 1
	cellarfeesKeeper.SetParams(ctx, params)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, sdk.NewInt(1000000))

	hooks := Hooks{k: cellarfeesKeeper}
	event := gravitytypes.SendToCosmosEvent{
		CosmosReceiver: feesAccount.GetAddress().String(),
		Amount:         sdk.NewInt(1),
		EthereumSender: "0x0000000000000000000000000000000000000000",
		TokenContract:  "0x1111111111111111111111111111111111111111",
	}
	cellarID := common.HexToAddress(event.EthereumSender)
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.FeeAccrualCounters{
		Counters: []cellarfeesTypes.FeeAccrualCounter{
			{
				Denom: gravityFeeDenom,
				Count: 0,
			},
		},
	})
	expectedCounters := cellarfeesTypes.FeeAccrualCounters{
		Counters: []cellarfeesTypes.FeeAccrualCounter{
			{
				Denom: gravityFeeDenom,
				Count: 1,
			},
		},
	}

	suite.corkKeeper.EXPECT().HasCellarID(ctx, cellarID).Return(true)
	suite.gravityKeeper.EXPECT().ERC20ToDenomLookup(ctx, common.HexToAddress(event.TokenContract)).Return(false, gravityFeeDenom)
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	require.NotPanics(func() { hooks.AfterSendToCosmosEvent(ctx, event) })

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(gravityFeeDenom, sdk.NewInt(0)))

	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), gravityFeeDenom).Return(sdk.NewCoin(gravityFeeDenom, event.Amount))

	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, event.Amount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)
	suite.auctionKeeper.EXPECT().GetActiveAuctions(ctx).Return([]*auctionTypes.Auction{})
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), gravityFeeDenom).Return(sdk.NewCoin(gravityFeeDenom, event.Amount))
	suite.auctionKeeper.EXPECT().BeginAuction(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })

	require.Equal(expectedCounters, cellarfeesKeeper.GetFeeAccrualCounters(ctx))

}

func (suite *KeeperTestSuite) TestAuctionBeginWithSufficientFundsWrongBlockHeight() {
	ctx, cellarfeesKeeper := suite.ctx, suite.cellarfeesKeeper
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.DefaultFeeAccrualCounters())
	suite.SetupHooksTests(ctx, cellarfeesKeeper)

	require := suite.Require()
	params := cellarfeesTypes.DefaultParams()
	params.AuctionInterval = 1000
	cellarfeesKeeper.SetParams(ctx, params)
	cellarfeesKeeper.SetLastRewardSupplyPeak(ctx, sdk.NewInt(1000000))

	hooks := Hooks{k: cellarfeesKeeper}
	event := gravitytypes.SendToCosmosEvent{
		CosmosReceiver: feesAccount.GetAddress().String(),
		Amount:         sdk.NewInt(2),
		EthereumSender: "0x0000000000000000000000000000000000000000",
		TokenContract:  "0x1111111111111111111111111111111111111111",
	}
	cellarID := common.HexToAddress(event.EthereumSender)
	cellarfeesKeeper.SetFeeAccrualCounters(ctx, cellarfeesTypes.FeeAccrualCounters{
		Counters: []cellarfeesTypes.FeeAccrualCounter{
			{
				Denom: gravityFeeDenom,
				Count: 1,
			},
		},
	})
	expectedCounters := cellarfeesTypes.FeeAccrualCounters{
		Counters: []cellarfeesTypes.FeeAccrualCounter{
			{
				Denom: gravityFeeDenom,
				Count: 2,
			},
		},
	}

	suite.corkKeeper.EXPECT().HasCellarID(ctx, cellarID).Return(true)
	suite.gravityKeeper.EXPECT().ERC20ToDenomLookup(ctx, common.HexToAddress(event.TokenContract)).Return(false, gravityFeeDenom)
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	require.NotPanics(func() { hooks.AfterSendToCosmosEvent(ctx, event) })

	// mocks
	suite.accountKeeper.EXPECT().GetModuleAccount(ctx, cellarfeesTypes.ModuleName).Return(feesAccount)

	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), appParams.BaseCoinUnit).Return(sdk.NewCoin(gravityFeeDenom, sdk.NewInt(0)))

	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), gravityFeeDenom).Return(sdk.NewCoin(gravityFeeDenom, event.Amount))

	expectedEmission := sdk.NewCoin(appParams.BaseCoinUnit, event.Amount)
	suite.bankKeeper.EXPECT().SendCoinsFromModuleToModule(ctx, gomock.Any(), gomock.Any(), sdk.NewCoins(expectedEmission)).Times(1)
	suite.auctionKeeper.EXPECT().GetActiveAuctions(ctx).Return([]*auctionTypes.Auction{})
	suite.bankKeeper.EXPECT().GetBalance(ctx, feesAccount.GetAddress(), gravityFeeDenom).Return(sdk.NewCoin(gravityFeeDenom, event.Amount))

	require.NotPanics(func() { cellarfeesKeeper.BeginBlocker(ctx) })

	require.Equal(expectedCounters, cellarfeesKeeper.GetFeeAccrualCounters(ctx))

}
