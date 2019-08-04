package cross

import (
	// "fmt"
	"github.com/classzz/classzz/wire"
	"github.com/classzz/classzz/txscript"
	"github.com/classzz/classzz/czzec"
	"github.com/classzz/classzz/chaincfg"
	"github.com/classzz/czzutil"
)

type EntangleTxInfo struct {
	Height uint64
	ExtTxHash []byte
}

func MakeEntangleTx(params *chaincfg.Params,inputs []*wire.TxIn,feeRate,inAmount czzutil.Amount,
	changeAddr czzutil.Address,info *EntangleTxInfo) (*wire.MsgTx, error) {
	// make pay script info include txHash and height
	scriptInfo,err := txscript.EntangleScript(info.Height,info.ExtTxHash)
	if err != nil {
		return nil, err
	}
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(&wire.TxOut{
		Value:    0,
		PkScript: scriptInfo,
	})
	var outputAmt czzutil.Amount = 0
	const (
		// spendSize is the largest number of bytes of a sigScript
		// which spends a p2pkh output: OP_DATA_73 <sig> OP_DATA_33 <pubkey>
		spendSize = 1 + 73 + 1 + 33
	)

	var (
		amtSelected  czzutil.Amount
		txSize       int
	)
	for _,in := range inputs {
		tx.AddTxIn(in)
		txSize = tx.SerializeSize() + spendSize*len(tx.TxIn)
	}
	reqFee := czzutil.Amount(txSize * int(feeRate))
	changeVal := amtSelected - outputAmt - reqFee

	if changeVal > 0{
		pkScript, err := txscript.PayToAddrScript(changeAddr)
		if err != nil {
			return nil, err
		}
		changeOutput := &wire.TxOut{
			Value:    int64(changeVal),
			PkScript: pkScript,
		}
		tx.AddTxOut(changeOutput)
	}

	return tx,nil
}

func SignEntangleTx(tx *wire.MsgTx,inputAmount []czzutil.Amount,
	priv *czzec.PrivateKey) error {

	for i, txIn := range tx.TxIn {	
		sigScript, err := txscript.SignatureScript(tx, i,
			int64(inputAmount[i].ToUnit(czzutil.AmountSatoshi)), nil,
			txscript.SigHashAll, priv, true)
		if err != nil {
			return err
		}
		txIn.SignatureScript = sigScript
	}
	
	return nil
}
