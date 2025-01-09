package main

import (
	"context"
	"math/big"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/test/operations"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	const (
		mainnetID uint32 = 0
		rollup1ID uint32 = 1
		rollup2ID uint32 = 2
	)
	ctx := context.Background()
	opsman1, err := operations.GetOpsman(ctx, "http://localhost:8123", "test_db", "8080", "9090", "5435", 1)
	if err != nil {
  	log.Fatal("Error: ", err)
  }
	opsman2, err := operations.GetOpsman(ctx, "http://localhost:8124", "test_db", "8080", "9090", "5435", 2)
	if err != nil {
  	log.Fatal("Error: ", err)
  }

	// Fund L2 sequencer for rollup 2. This is super dirty, but have no better way to do this at the moment
	polAddr := common.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3")
	rollup2Sequencer := common.HexToAddress("0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65")
	polAmount, _ := big.NewInt(0).SetString("10000000000000000000000", 10)
	err = opsman2.MintPOL(ctx, polAddr, polAmount, operations.L1)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	err = opsman2.ERC20Transfer(ctx, polAddr, rollup2Sequencer, polAmount, operations.L1)
	if err != nil {
    log.Fatal("Error: ", err)
  }

	// L1 and R1 interactions
	log.Info("L1 -- eth --> R1")
	bridge(ctx, opsman1, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(999999999999999999),
	})

	log.Info("R1 -- eth --> L1")
	bridge(ctx, opsman1, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: common.Address{},
		amount:          big.NewInt(42069),
	})

	l1TokenAddr, _, err := opsman1.DeployNonERC20(ctx, "CREATED ON L1", "CL1", operations.L1)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	err = opsman1.MintNonERC20(ctx, l1TokenAddr, big.NewInt(999999999999999999), operations.L1)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	log.Info("L1 -- token from L1 --> R1")
	bridge(ctx, opsman1, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("R1 -- token from L1 --> L1")
	bridge(ctx, opsman1, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  mainnetID,
		originTokenAddr: l1TokenAddr,
		amount:          big.NewInt(42069),
	})

	rollup1TokenAddr, _, err := opsman1.DeployNonERC20(ctx, "CREATED ON Rollup 1", "CR1", operations.L2)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	err = opsman1.MintNonERC20(ctx, rollup1TokenAddr, big.NewInt(999999999999999999), operations.L2)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	log.Info("R1 -- token from R1 --> L1")
	bridge(ctx, opsman1, bridgeData{
		originNet:       rollup1ID,
		destNet:         mainnetID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

	log.Info("L1 -- token from R1 --> R1")
	bridge(ctx, opsman1, bridgeData{
		originNet:       mainnetID,
		destNet:         rollup1ID,
		originTokenNet:  rollup1ID,
		originTokenAddr: rollup1TokenAddr,
		amount:          big.NewInt(42069),
	})

}

type bridgeData struct {
	originNet       uint32
	destNet         uint32
	originTokenNet  uint32
	originTokenAddr common.Address
	amount          *big.Int
}

func bridge(
	ctx context.Context,
	opsman *operations.Manager,
	bd bridgeData,
) {
	var (
		// This addresses are hardcoded on opsman. Would be nice to make it more flexible
		// to be able to operate multiple accounts
		destAddr common.Address
		l1Addr   = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
		l2Addr   = common.HexToAddress("0xc949254d682d8c9ad5682521675b8f43b102aec4")
	)
	if bd.destNet == 0 {
		destAddr = l1Addr
	} else {
		destAddr = l2Addr
	}
	initialL1Balance, initialL2Balance, err := opsman.GetBalances(ctx, uint32(bd.originTokenNet), bd.originTokenAddr, l1Addr, l2Addr)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	log.Debugf("initial balance on L1: %d, initial balance on L2: %d", initialL1Balance.Int64(), initialL2Balance.Int64())
	if bd.originNet == 0 {
		tokenAddr, err := opsman.GetTokenAddress(ctx, operations.L1, bd.originTokenNet, bd.originTokenAddr)
		if err != nil {
      log.Fatal("Error: ", err)
    }
		log.Debugf("depositing %d tokens of addr %s on L1 to network %d", bd.amount.Uint64(), tokenAddr, bd.destNet)
		err = opsman.SendL1Deposit(ctx, tokenAddr, bd.amount, uint32(bd.destNet), &destAddr)
		if err != nil {
      log.Fatal("Error: ", err)
    }
	} else {
		tokenAddr, err := opsman.GetTokenAddress(ctx, operations.L2, bd.originTokenNet, bd.originTokenAddr)
		if err != nil {
      log.Fatal("Error: ", err)
    }
		log.Debugf("depositing %d tokens of addr %s to Network %d", bd.amount.Uint64(), tokenAddr, bd.destNet)
		err = opsman.SendL2Deposit(ctx, tokenAddr, bd.amount, uint32(bd.destNet), &destAddr, operations.L2)
		if err != nil {
      log.Fatal("Error: ", err)
    }
	}
	log.Debug("deposit sent")

	log.Debug("checking deposits from bridge service...")
	deposits, err := opsman.GetBridgeInfoByDestAddr(ctx, &destAddr)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	deposit := deposits[0]

	if bd.originNet == 0 {
		log.Debug("waiting for claim tx to be sent on behalf of the user by bridge service...")
		err = opsman.CheckClaim(ctx, deposit)
		if err != nil {
      log.Fatal("Error: ", err)
    }
		log.Debug("deposit claimed on L2")
	} else {
		log.Debug("getting proof to perform claim from bridge service...")
		smtProof, smtRollupProof, globaExitRoot, err := opsman.GetClaimData(ctx, deposit.NetworkId, deposit.DepositCnt)
		if err != nil {
      log.Fatal("Error: ", err)
    }
		log.Debug("sending claim tx to L1")
		err = opsman.SendL1Claim(ctx, deposit, smtProof, smtRollupProof, globaExitRoot)
		if err != nil {
      log.Fatal("Error: ", err)
    }
		log.Debug("claim sent")
	}
	time.Sleep(2 * time.Second)

	afterClaimL1Balance, afterClaimL2Balance, err := opsman.GetBalances(ctx, uint32(bd.originTokenNet), bd.originTokenAddr, l1Addr, l2Addr)
	if err != nil {
    log.Fatal("Error: ", err)
  }
	log.Debugf("deposit claimed on network %d. final balance on L1: %d, final balance on L2: %d", bd.originNet, afterClaimL1Balance.Int64(), afterClaimL2Balance.Int64())
}
