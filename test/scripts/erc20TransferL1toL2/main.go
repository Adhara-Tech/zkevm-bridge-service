package main

import (
	"context"
	"errors"
	"math/big"
	"net/http"
	"fmt"
	"io"
	"time"

  "github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
  "github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	erc20 "github.com/0xPolygonHermez/zkevm-node/etherman/smartcontracts/pol"
	"github.com/0xPolygonHermez/zkevm-node/test/operations"
	"google.golang.org/protobuf/encoding/protojson"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	l1BridgeAddr       = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
  l2BridgeAddr       = "0xFe12ABaa190Ef0c8638Ee0ba9F828BF41368Ca0E"
	accHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	accHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l1NetworkURL       = "http://localhost:8545"
	l2NetworkURL       = "http://localhost:8123"
	bridgeURL          = "http://localhost:8080"

	funds              = 90000000000000000 // nolint
	destNetwork uint32 = 1
	miningTimeout      = 180
)

const (
	// Typically the time to auto-claim is 15min (L1->L2)
	maxTimeToAutoClaim = 60 * time.Minute
	// Typically the time to claim a deposit is 1 hours (L2 -> L1)
	maxTimeToClaimReady   = 120 * time.Minute
	timeBetweenCheckClaim = 60 * time.Second
	mtHeight              = 32
)

func main() {
	ctx := context.Background()
	l1client, err := utils.NewClient(ctx, l1NetworkURL, common.HexToAddress(l1BridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
	l1auth, err := l1client.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

  l2client, err := utils.NewClient(ctx, l2NetworkURL, common.HexToAddress(l2BridgeAddr))
	if err != nil {
		log.Fatal("Error: ", err)
	}
  l2auth, err := l2client.GetSigner(ctx, accHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}

  log.Infof("ERC20 [L1->L2] Deploy token")
	tokenAddr, err := deployToken(ctx, l1client, l1auth, common.HexToAddress(l1BridgeAddr))
	if err != nil {
  	log.Fatal("Error: ", err)
  }
	log.Infof("ERC20 [L1->L2] L1: Token Addr: ", tokenAddr.Hex())

	log.Infof("ERC20 [L1->L2] assetERC20L1ToL2")
	amount := big.NewInt(funds)
	destAddr := common.HexToAddress(accHexAddress)
	log.Info("Sending bridge tx...")
	tx, err := sendBridgeAsset(ctx, l1client, tokenAddr, amount, destNetwork, &destAddr, []byte{}, l1auth)
	if err != nil {
		log.Fatal("Error: ", err)
	}
  err = operations.WaitTxToBeMined(ctx, l1client, tx, miningTimeout*time.Second)
  if err != nil {
  	log.Fatal("error: ", err)
  }
  deposit, err := waitDepositToBeReadyToClaim(ctx, bridgeURL, tx.Hash(), maxTimeToClaimReady, destAddr.String())
  if err != nil {
  	log.Fatal("Error: ", err)
  }
  log.Info(deposit)
  err = waitToAutoClaimTx(ctx, l2client, deposit, 60*time.Second)
  if err != nil {
  	log.Errorf("ERC20 [L1->L2] Doing manual claim")
  	err = manualClaimDepositL2(ctx, l2auth, l2client, bridgeURL, deposit)
  	if err != nil {
      log.Fatal("Error: ", err)
     }
  }
	log.Info("Success!")
}

func deployToken(ctx context.Context, client *utils.Client, auth *bind.TransactOpts, bridgeAddr common.Address) (common.Address, error) {
	tokenAddr, _, err := client.DeployERC20(ctx, "A COIN", "ACO", auth)
	if err != nil {
		return tokenAddr, err
	}
	log.Info("Token Addr: ", tokenAddr.Hex())
	amountTokens := new(big.Int).SetUint64(1000000000000000000)
	err = client.ApproveERC20(ctx, tokenAddr, bridgeAddr, amountTokens, auth)
	if err != nil {
		return tokenAddr, err
	}
	err = client.MintERC20(ctx, tokenAddr, amountTokens, auth)
	if err != nil {
		return tokenAddr, err
	}
	erc20Balance, err := getAccountTokenBalance(ctx, auth, client, tokenAddr, nil)
	if err != nil {
		return tokenAddr, err
	}
	log.Info("ERC20 Balance: ", erc20Balance.String())
	return tokenAddr, nil
}

func getAccountTokenBalance(ctx context.Context, auth *bind.TransactOpts, client *utils.Client, tokenAddr common.Address, account *common.Address) (*big.Int, error) {

	if account == nil {
		account = &auth.From
	}
	erc20Token, err := erc20.NewPol(tokenAddr, client)
	if err != nil {
		return big.NewInt(0), nil
	}
	balance, err := erc20Token.BalanceOf(&bind.CallOpts{Pending: false}, *account)
	if err != nil {
		return big.NewInt(0), nil
	}
	return balance, nil
}

func sendBridgeAsset(ctx context.Context, c *utils.Client, tokenAddr common.Address, amount *big.Int, destNetwork uint32,
	destAddr *common.Address, metadata []byte, auth *bind.TransactOpts) (*types.Transaction, error) {
	emptyAddr := common.Address{}
	if tokenAddr == emptyAddr {
		auth.Value = amount
	}
	if destAddr == nil {
		destAddr = &auth.From
	}
	tx, err := c.Bridge.BridgeAsset(auth, destNetwork, *destAddr, amount, tokenAddr, true, metadata)
	if err != nil {
		log.Error("error sending deposit. Error: ", err)
		return nil, err
	}
	return tx, err
}

func waitDepositToBeReadyToClaim(ctx context.Context, bridgeURL string, assetTxHash common.Hash, timeout time.Duration, destAddr string) (*pb.Deposit, error) {
	startTime := time.Now()
	for true {
		log.Info("Waiting to deposit (", destAddr, ") fo assetTx: ", assetTxHash.Hex(), "...")
	  resp, err := http.Get(fmt.Sprintf("%s%s/%s?offset=%d&limit=%d", bridgeURL, "/bridges", destAddr, 0, 100))
  	if err != nil {
  		log.Fatal("Error: ", err)
  	}
    bodyBytes, err := io.ReadAll(resp.Body)
  	if err != nil {
  		log.Fatal("Error: ", err)
  	}
  	var bridgeResp pb.GetBridgesResponse
  	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
  	if err != nil {
  		log.Fatal("Error: ", err)
  	}
  	deposits := bridgeResp.Deposits
		if err != nil {
			return nil, err
		}

		for _, deposit := range deposits {
			depositHash := common.HexToHash(deposit.TxHash)
			if depositHash == assetTxHash {
				log.Info("Deposit found: ", deposit, " ready:", deposit.ReadyForClaim)
				if deposit.ReadyForClaim {
					log.Info("Found claim! Claim Is ready  Elapsed time: ", time.Since(startTime))
					return deposit, nil
				}
			}
		}
		if time.Since(startTime) > timeout {
			return nil, fmt.Errorf("Timeout waiting for deposit to be ready to be claimed")
		}
		log.Info("Sleeping ", timeBetweenCheckClaim.String(), "Elapsed time: ", time.Since(startTime))
		time.Sleep(timeBetweenCheckClaim)
	}
	return nil, nil
}

func waitToAutoClaimTx(ctx context.Context, c *utils.Client, deposit *pb.Deposit, timeout time.Duration) error {
	startTime := time.Now()
	emptyHash := common.Hash{}
	claimTxHash := common.HexToHash(deposit.ClaimTxHash)
	if claimTxHash != emptyHash {
		log.Debugf("Found claim! Claim Tx Hash: ", claimTxHash.Hex())
		// The claim from L1 -> L2 is done by the bridge service to L2
		receipt, err := waitTxToBeMinedByTxHash(ctx, c, claimTxHash, 60*time.Second)
		if err != nil {
			return err
		}
		log.Debug("Receipt: ", receipt, " Elapsed time: ", time.Since(startTime))
		return nil
	}
	return fmt.Errorf("No auto-claim tx, deposit: %+v", deposit)
}

func waitTxToBeMinedByTxHash(parentCtx context.Context, client *utils.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()
	receipt, err := waitMinedByTxHash(ctx, client, txHash)
	if errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	} else if err != nil {
		log.Errorf("error waiting tx %s to be mined: %w", txHash, err)
		return nil, err
	}
	if receipt.Status == types.ReceiptStatusFailed {
		reason := " reverted "
		return nil, fmt.Errorf("transaction has failed, reason: %s, receipt: %+v. txHash:%s", reason, receipt, txHash.Hex())
	}
	log.Debug("Transaction successfully mined: ", txHash)
	return receipt, nil
}

func waitMinedByTxHash(ctx context.Context, client *utils.Client, txHash common.Hash) (*types.Receipt, error) {
	queryTicker := time.NewTicker(time.Second)
	defer queryTicker.Stop()

	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		if errors.Is(err, ethereum.NotFound) {
			log.Debug("Transaction not yet mined")
		} else {
			log.Debug("Receipt retrieval failed", "err", err)
		}

		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}

func manualClaimDepositL2(ctx context.Context, auth *bind.TransactOpts, c *utils.Client, bridgeURL string, deposit *pb.Deposit) error {
  resp, err := http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", bridgeURL, "/merkle-proof", deposit.NetworkId, deposit.DepositCnt))
	if err != nil {
  	log.Fatal("error: ", err)
  }
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
  	log.Fatal("error: ", err)
  }
	var proofResp pb.GetProofResponse
	err = protojson.Unmarshal(bodyBytes, &proofResp)
	if err != nil {
  	log.Fatal("error: ", err)
  }
	proof := proofResp.Proof

	log.Debug("deposit: ", deposit)
	log.Debug("mainnetExitRoot: ", proof.MainExitRoot)
	log.Debug("rollupExitRoot: ", proof.RollupExitRoot)

	smtProof := convertMerkleProof(proof.MerkleProof)
	printMerkleProof(smtProof, "smtProof: ")
	smtRollupProof := convertMerkleProof(proof.RollupMerkleProof)
	printMerkleProof(smtRollupProof, "smtRollupProof: ")

	ger := &etherman.GlobalExitRoot{
		ExitRoots: []common.Hash{common.HexToHash(proof.MainExitRoot), common.HexToHash(proof.RollupExitRoot)},
	}

	log.Infof(" L2.Claim()")
	err = c.SendClaim(ctx, deposit, smtProof, smtRollupProof, ger, auth)

	return err
}

func convertMerkleProof(mkProof []string) [mtHeight][32]byte {
	var smtProof [mtHeight][32]byte
	for i := 0; i < len(mkProof); i++ {
		smtProof[i] = common.HexToHash(mkProof[i])
	}
	return smtProof
}

func printMerkleProof(mkProof [mtHeight][32]byte, title string) {
	for i := 0; i < len(mkProof); i++ {
		fmt.Println(title, "[", i, "]", mkProof[i])
	}
}



