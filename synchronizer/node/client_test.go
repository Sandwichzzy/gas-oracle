package node

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBlockReceiptsByNumber(t *testing.T) {
	testRpcs := [5]string{"https://eth-sepolia.g.alchemy.com/v2/ejmFf9C-9RAwjDv13Y1fVSUoUN9tm2sh", "https://opt-sepolia.g.alchemy.com/v2/ejmFf9C-9RAwjDv13Y1fVSUoUN9tm2sh", "https://arb-sepolia.g.alchemy.com/v2/ejmFf9C-9RAwjDv13Y1fVSUoUN9tm2sh", "https://base-sepolia.g.alchemy.com/v2/ejmFf9C-9RAwjDv13Y1fVSUoUN9tm2sh", "https://rpc.cpchain.com"}
	for i := 0; i < len(testRpcs); i++ {
		fmt.Printf("rpc %s\n", testRpcs[i])

		rpc := testRpcs[i]
		client, err := DialEthClient(context.Background(), rpc)
		if err != nil {
			t.Error("DialEthClient ", rpc, " failed")
		}

		number, err := client.GetLatestBlock(context.Background())
		if err != nil {
			t.Error("GetLatestBlock ", rpc, " failed")
		}
		txs, _, err := client.BlockDetailByNumber(context.Background(), number)
		if err != nil {
			t.Error("BlockDetailByNumber ", rpc, " failed")
		}
		blockReceipts, err := client.BlockReceiptsByNumber(context.Background(), number)
		if err != nil {
			t.Error("BlockReceiptsByNumber ", rpc, " failed")
		}
		if len(txs) != len(blockReceipts) {
			t.Error("txs.len != receipts.len ", len(txs), len(blockReceipts))
		}
		println("blocknumber", number.String(), "len(txs)", len(txs))
		for i := 0; i < len(blockReceipts); i++ {
			// fmt.Printf("receipts %+v %+v", i, blockReceipts[i])
		}

		if len(blockReceipts) > 0 {
			for i := 0; i < 1; i++ {
				tx := txs[i]
				receipt, err := client.TxReceiptDetailByHash(context.Background(), common.HexToHash(tx))
				if err != nil {
					t.Error("TxReceiptDetailByHash ", tx, " failed")
				}

				// fmt.Printf("receipt %+v", receipt)
				// fmt.Printf("receipt %+v", blockReceipts[0])

				if receipt.EffectiveGasPrice.Cmp(blockReceipts[i].EffectiveGasPrice) != 0 {
					t.Error("tx.EffectiveGasPrice not equal", tx)
				}
			}
		}

	}
}
