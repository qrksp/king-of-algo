package integration

import (
	"context"
	"testing"
	"time"

	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/qrksp/king-of-algo/client"
	. "github.com/smartystreets/goconvey/convey"
)

func TestContractDeployment(t *testing.T) {
	Convey("Contract deployment", t, func() {
		s := NewSuite()

		owner := s.Accounts[0]
		Convey("Creates app and sets default state", func() {
			appID, err := client.Deploy(context.Background(), s.Algod, owner, time.Hour, "")
			So(err, ShouldBeNil)

			state, err := client.GetContractState(context.Background(), s.Algod, owner, appID)
			So(err, ShouldBeNil)

			So(state.EndOfReign.UTC(), ShouldHappenAfter, time.Now().UTC().Add(time.Minute*30))
			So(state.EndOfReign.UTC(), ShouldHappenBefore, time.Now().UTC().Add(time.Hour))
			So(state.InitPrice, ShouldEqual, 100000)
			So(state.King, ShouldEqual, "")
			So(state.KingPrice, ShouldEqual, 100000)
			So(state.Admin, ShouldEqual, owner.Address.String())
			So(state.AdminFee, ShouldEqual, 5)
			So(state.RewardMultiplier, ShouldEqual, 75)
		})
	})
}

func TestBecomeKing(t *testing.T) {
	Convey("client.BecomeKing()", t, func() {
		s := NewSuite()

		owner := s.Accounts[0]
		first := s.Accounts[1]
		second := s.Accounts[2]

		period := time.Second * 35 // TODO: this is tricky, maybe I should remove the third king

		// This worked to debug the test.
		// go func() {
		// 	<-time.Tick(period)
		// 	fmt.Println("times up!")
		// }()

		appID, err := client.Deploy(context.Background(), s.Algod, owner, period, "")
		So(err, ShouldBeNil)

		Convey("Become first king when there is no previous king", func() {
			state, _ := client.GetContractState(context.Background(), s.Algod, owner, appID)

			beforeBalances := s.getAccountsBalances()
			priceToBeKing := state.KingPrice

			_, err := client.BecomeKing(
				context.Background(),
				s.Algod,
				false,
				client.NewBecomeKingParams(
					s.getSuggestedParams(),
					appID,
					state,
					first,
					"I am the first king",
				),
				3,
			)
			So(err, ShouldBeNil)

			state, err = client.GetContractState(context.Background(), s.Algod, owner, appID)
			So(err, ShouldBeNil)

			So(state.EndOfReign, ShouldHappenBefore, time.Now().Add(period))
			So(state.InitPrice, ShouldEqual, 100000)
			So(state.King, ShouldEqual, first.Address.String())
			So(state.KingPrice, ShouldEqual, 200000)
			So(state.Admin, ShouldEqual, owner.Address.String())
			So(state.AdminFee, ShouldEqual, 5)
			So(state.RewardMultiplier, ShouldEqual, 75)

			balances := s.getAccountsBalances()

			So(balances[owner.Address.String()], ShouldEqual, beforeBalances[owner.Address.String()]+(multiplyPercentage(priceToBeKing, state.AdminFee)))
			So(balances[first.Address.String()], ShouldEqual, beforeBalances[first.Address.String()]-priceToBeKing-(transaction.MinTxnFee*3))
			So(balances[second.Address.String()], ShouldEqual, beforeBalances[second.Address.String()])

			Convey("Become second king", func() {
				state, _ := client.GetContractState(context.Background(), s.Algod, owner, appID)

				beforeBalances := s.getAccountsBalances()
				priceToBeKing := state.KingPrice

				So(state.EndOfReign, ShouldHappenAfter, time.Now())

				_, err := client.BecomeKing(
					context.Background(),
					s.Algod,
					false,
					client.NewBecomeKingParams(
						s.getSuggestedParams(),
						appID,
						state,
						second,
						"I am the second king",
					),
					3,
				)

				So(err, ShouldBeNil)

				state, err = client.GetContractState(context.Background(), s.Algod, owner, appID)
				So(err, ShouldBeNil)

				So(state.InitPrice, ShouldEqual, 100000)
				So(state.KingPrice, ShouldEqual, 400000)
				So(state.King, ShouldEqual, second.Address.String())
				So(state.Admin, ShouldEqual, owner.Address.String())
				So(state.AdminFee, ShouldEqual, 5)

				balances := s.getAccountsBalances()

				So(balances[owner.Address.String()], ShouldEqual, beforeBalances[owner.Address.String()]+(multiplyPercentage(priceToBeKing, state.AdminFee)))
				So(balances[second.Address.String()], ShouldEqual, beforeBalances[second.Address.String()]-priceToBeKing-(transaction.MinTxnFee*4))
				So(balances[first.Address.String()], ShouldEqual, beforeBalances[first.Address.String()]+(multiplyPercentage(priceToBeKing, state.RewardMultiplier)))

				Convey("Become third king", func() {
					state, _ := client.GetContractState(context.Background(), s.Algod, owner, appID)

					beforeBalances := s.getAccountsBalances()
					priceToBeKing := state.KingPrice

					So(state.EndOfReign, ShouldHappenAfter, time.Now())

					_, err := client.BecomeKing(
						context.Background(),
						s.Algod,
						false,
						client.NewBecomeKingParams(
							s.getSuggestedParams(),
							appID,
							state,
							first,
							"I am the third king",
						),
						3,
					)
					So(err, ShouldBeNil)

					state, err = client.GetContractState(context.Background(), s.Algod, owner, appID)
					So(err, ShouldBeNil)

					So(state.InitPrice, ShouldEqual, 100000)
					So(state.KingPrice, ShouldEqual, 800000)
					So(state.King, ShouldEqual, first.Address.String())
					So(state.Admin, ShouldEqual, owner.Address.String())
					So(state.AdminFee, ShouldEqual, 5)

					balances := s.getAccountsBalances()

					So(balances[owner.Address.String()], ShouldEqual, beforeBalances[owner.Address.String()]+(multiplyPercentage(priceToBeKing, state.AdminFee)))
					So(balances[first.Address.String()], ShouldEqual, beforeBalances[first.Address.String()]-priceToBeKing-(transaction.MinTxnFee*4))
					So(balances[second.Address.String()], ShouldEqual, beforeBalances[second.Address.String()]+(multiplyPercentage(priceToBeKing, state.RewardMultiplier)))

					Convey("Returns an error for unbalanced rewards exploit", func() {
						state, _ := client.GetContractState(context.Background(), s.Algod, owner, appID)

						beforeBalances := s.getAccountsBalances()

						endOfReign := state.EndOfReign
						timeLeft := endOfReign.Sub(time.Now())
						if timeLeft > 0 {
							time.Sleep(timeLeft + time.Second*10)
						}

						So(state.EndOfReign, ShouldHappenBefore, time.Now())

						_, err := client.BecomeKingUnbalancedRewardsExploit(
							context.Background(),
							s.Algod,
							false,
							client.NewBecomeKingParams(
								s.getSuggestedParams(),
								appID,
								state,
								second,
								"I am a hacker",
							),
							3,
						)
						So(err, ShouldNotBeNil)

						state, err = client.GetContractState(context.Background(), s.Algod, owner, appID)
						So(err, ShouldBeNil)

						So(state.InitPrice, ShouldEqual, 100000)
						So(state.KingPrice, ShouldEqual, 800000)
						So(state.King, ShouldEqual, first.Address.String())
						So(state.Admin, ShouldEqual, owner.Address.String())
						So(state.AdminFee, ShouldEqual, 5)

						balances := s.getAccountsBalances()

						So(balances[owner.Address.String()], ShouldEqual, beforeBalances[owner.Address.String()])
						So(balances[second.Address.String()], ShouldEqual, beforeBalances[second.Address.String()])
						So(balances[first.Address.String()], ShouldEqual, beforeBalances[first.Address.String()])
					})

					Convey("Become king after end of reign", func() {
						state, _ := client.GetContractState(context.Background(), s.Algod, owner, appID)

						beforeBalances := s.getAccountsBalances()
						beforeContractAccountInfo := s.getContractAccountInfo(appID)

						priceToBeKing := state.InitPrice

						endOfReign := state.EndOfReign
						timeLeft := endOfReign.Sub(time.Now())
						if timeLeft > 0 {
							time.Sleep(timeLeft + time.Second*10)
						}

						So(state.EndOfReign, ShouldHappenBefore, time.Now())

						_, err := client.BecomeKing(
							context.Background(),
							s.Algod,
							false,
							client.NewBecomeKingParams(
								s.getSuggestedParams(),
								appID,
								state,
								second,
								"I am the new king",
							),
							3,
						)
						So(err, ShouldBeNil)

						state, err = client.GetContractState(context.Background(), s.Algod, owner, appID)
						So(err, ShouldBeNil)

						So(state.InitPrice, ShouldEqual, 100000)
						So(state.KingPrice, ShouldEqual, 200000)
						So(state.King, ShouldEqual, second.Address.String())
						So(state.Admin, ShouldEqual, owner.Address.String())
						So(state.AdminFee, ShouldEqual, 5)

						balances := s.getAccountsBalances()

						afterContractBalance := s.getContractAccountInfo(appID)

						reward := multiplyPercentage(priceToBeKing, state.RewardMultiplier)
						adminFee := multiplyPercentage(priceToBeKing, state.AdminFee)
						comp := priceToBeKing - reward - adminFee

						So(balances[owner.Address.String()], ShouldEqual, beforeBalances[owner.Address.String()]+(adminFee))
						So(balances[second.Address.String()], ShouldEqual, beforeBalances[second.Address.String()]-priceToBeKing-(transaction.MinTxnFee*5))
						So(afterContractBalance.Amount, ShouldEqual, s.minBalance()+comp)
						So(balances[first.Address.String()], ShouldEqual, beforeBalances[first.Address.String()]+beforeContractAccountInfo.Amount-s.minBalance()+reward)
					})
				})
			})
		})
	})
}
