package polka

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Tx struct {
	metadata *types.Metadata
}

func NewTx(metadataString string) (*Tx, error) {
	var metadata types.Metadata
	if err := types.DecodeFromHexString(metadataString, &metadata); err != nil {
		return nil, err
	}
	return &Tx{
		metadata: &metadata,
	}, nil
}

func (c *Chain) GetTx() (*Tx, error) {
	client, err := getConnectedPolkaClient(c.RpcUrl)
	if err != nil {
		return nil, err
	}
	metadata, err := client.MetadataString()
	if err != nil {
		return nil, err
	}
	return NewTx(metadata)
}

type Transaction struct {
	extrinsic *types.Extrinsic
}

func (t *Transaction) GetSignData(genesisHashString string, nonce int64, specVersion, transVersion int32) ([]byte, error) {
	var methodBytes []byte
	genesisHash, err := types.NewHashFromHexString(genesisHashString)
	if err != nil {
		return nil, err
	}
	if t.extrinsic == nil {
		return nil, ErrNilExtrinsic
	}
	t.extrinsic.Signature = types.ExtrinsicSignatureV4{
		Nonce: types.NewUCompactFromUInt(uint64(nonce)),
		Era:   types.ExtrinsicEra{IsImmortalEra: true},
		Tip:   types.NewUCompactFromUInt(0),
	}
	methodBytes, err = types.EncodeToBytes(t.extrinsic.Method)
	if err != nil {
		return nil, err
	}
	return types.EncodeToBytes(types.ExtrinsicPayloadV4{
		ExtrinsicPayloadV3: types.ExtrinsicPayloadV3{
			Method:      methodBytes,
			Era:         t.extrinsic.Signature.Era,
			Nonce:       t.extrinsic.Signature.Nonce,
			Tip:         t.extrinsic.Signature.Tip,
			SpecVersion: types.NewU32(uint32(specVersion)),
			GenesisHash: genesisHash,
			BlockHash:   genesisHash,
		},
		TransactionVersion: types.NewU32(uint32(transVersion)),
	})

}

func (t *Transaction) GetUnSignTx() (string, error) {
	if t.extrinsic == nil {
		return "", ErrNilExtrinsic
	}
	return types.EncodeToHexString(t.extrinsic)
}

func (t *Transaction) GetTxFromHex(signerPublicKeyHex string, signatureDataHex string) (string, error) {
	signerPublicKey, err := types.HexDecodeString(signerPublicKeyHex)
	if err != nil {
		return "", err
	}
	signatureData, err := types.HexDecodeString(signatureDataHex)
	if err != nil {
		return "", err
	}
	return t.GetTx(signerPublicKey, signatureData)
}

func (t *Transaction) GetTx(signerPublicKey []byte, signatureData []byte) (string, error) {
	if signatureData == nil {
		return "", ErrNotSigned
	}

	if signerPublicKey == nil {
		return "", ErrNoPublicKey
	}

	t.extrinsic.Signature.Signer = types.NewMultiAddressFromAccountID(signerPublicKey)
	t.extrinsic.Signature.Signature = types.MultiSignature{IsSr25519: true, AsSr25519: types.NewSignature(signatureData)}
	t.extrinsic.Version |= types.ExtrinsicBitSigned
	return types.EncodeToHexString(t.extrinsic)
}

func (t *Tx) NewTransactionFromHex(txHex string) (*Transaction, error) {
	var (
		transaction = &Transaction{}
	)

	if t.metadata == nil {
		return nil, ErrNilMetadata
	}

	transaction.extrinsic = &types.Extrinsic{}
	err := types.DecodeFromHexString(txHex, &transaction.extrinsic)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (t *Tx) NewXGatewayBitcoinCreateTaprootWithdrawTx(ids, transactionHex string) (*Transaction, error) {
	idList := strings.Split(ids, ",")
	idsU32 := make([]uint32, 0)
	for _, v := range idList {
		if len(v) == 0 {
			continue
		}
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return nil, err
		}
		idsU32 = append(idsU32, uint32(id))
	}
	transactionBytes, err := types.HexDecodeString(transactionHex)
	if err != nil {
		return nil, err
	}
	return t.NewExtrinsics("XGatewayBitcoin.create_taproot_withdraw_tx", idsU32, types.NewBytes(transactionBytes))
}

func (t *Tx) NewExtrinsics(call string, args ...interface{}) (*Transaction, error) {
	if t.metadata == nil {
		return nil, ErrNilMetadata
	}
	var (
		transaction = &Transaction{}
	)

	callType, err := types.NewCall(t.metadata, call, args...)
	if err != nil {
		return nil, err
	}
	extrinsic := types.NewExtrinsic(callType)
	transaction.extrinsic = &extrinsic
	return transaction, nil
}

func (t *Tx) NewBalanceTransferTx(dest, amount string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, ErrNumber
	}
	destAccountID, err := addressStringToMultiAddress(dest)
	if err != nil {
		return nil, err
	}
	return t.NewExtrinsics("Balances.transfer", destAccountID, types.NewUCompact(amountBigint))
}

func (t *Tx) NewComingNftTransferTx(dest string, cid int64) (*Transaction, error) {
	destAccountID, err := addressStringToMultiAddress(dest)
	if err != nil {
		return nil, err
	}
	return t.NewExtrinsics("ComingNFT.transfer", types.NewU64(uint64(cid)), destAccountID)
}

func (t *Tx) NewStakingBondTx(controller, value, payee string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, ErrNumber
	}
	controllerAccountID, err := addressStringToMultiAddress(controller)
	if err != nil {
		return nil, err
	}

	return t.NewExtrinsics("Staking.bond", controllerAccountID, types.NewUCompact(amountBigint), byte(1))
}

func (t *Tx) NewStakingBondExtraTx(value string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, ErrNumber
	}

	return t.NewExtrinsics("Staking.bond_extra", types.NewUCompact(amountBigint))
}

func (t *Tx) NewPayoutStakesTx(address string, era uint32) (*Transaction, error) {
	// ss58Format := base58.Decode(address)
	// if len(ss58Format) == 0 {
	// 	return nil, errors.New("The address ss58 format is invalid")
	// }
	// pubkey, err := hex.DecodeString(ss58.Decode(address, int(ss58Format[0])))
	// if err != nil {
	// 	return nil, err
	// }
	validate, err := addressStringToMultiAddress(address)
	if err != nil {
		return nil, err
	}
	return t.NewExtrinsics("Staking.payout_stakers", validate.AsID, era)
}

func (t *Tx) NewUnBoundTx(amount string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, ErrNumber
	}
	return t.NewExtrinsics("Staking.unbond", types.NewUCompact(amountBigint))
}

func (t *Tx) NewWithDrawUnbondTx(span uint32) (*Transaction, error) {
	return t.NewExtrinsics("Staking.withdraw_unbonded", span)
}

func (t *Tx) NewBondAndNominateTx(stash, controller, value string, targets []string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, ErrNumber
	}

	stashAccountID, err := addressStringToMultiAddress(stash)
	if err != nil {
		return nil, err
	}
	controllerAccountID, err := addressStringToMultiAddress(controller)
	if err != nil {
		return nil, err
	}
	// payeeAccountID, err := addressStringToMultiAddress(payee)
	// if err != nil {
	// 	return nil, err
	// }
	Call1, err := types.NewCall(t.metadata, "Staking.bond", stashAccountID, types.NewUCompact(amountBigint), byte(1))
	if err != nil {
		return nil, err
	}

	nomiTargets := make([]types.MultiAddress, 0)
	for i := 0; i < len(targets); i++ {
		target, err := addressStringToMultiAddress(targets[i])
		if err != nil {
			return nil, err
		}
		nomiTargets = append(nomiTargets, target)
	}

	Call2, err := types.NewCall(t.metadata, "Staking.nominate", nomiTargets)
	if err != nil {
		return nil, err
	}

	if stash != controller {
		call3, err := types.NewCall(t.metadata, "Staking.set_controller", controllerAccountID)
		if err != nil {
			return nil, err
		}

		return t.NewExtrinsics("Utility.batch_all", []types.Call{Call1, Call2, call3})
	}
	return t.NewExtrinsics("Utility.batch_all", []types.Call{Call1, Call2})

}

func (t *Tx) NewChillAndUnbondTx(amount string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, ErrNumber
	}
	Call1, err := types.NewCall(t.metadata, "Staking.chill")
	if err != nil {
		return nil, err
	}
	Call2, err := types.NewCall(t.metadata, "Staking.unbond", types.NewUCompact(amountBigint))
	if err != nil {
		return nil, err
	}
	return t.NewExtrinsics("Utility.batch_all", []types.Call{Call1, Call2})
}

func (t *Tx) NewNominateTx(targets []string) (*Transaction, error) {
	nomiTargets := make([]types.MultiAddress, 0)
	for i := 0; i < len(targets); i++ {
		target, err := addressStringToMultiAddress(targets[i])
		if err != nil {
			return nil, err
		}
		nomiTargets = append(nomiTargets, target)
	}

	return t.NewExtrinsics("Staking.nominate", nomiTargets)
}

func (t *Tx) NewXAssetsTransferTx(dest, amount string) (*Transaction, error) {
	amountBigint, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		return nil, ErrNumber
	}
	destAccountID, err := addressStringToMultiAddress(dest)
	if err != nil {
		return nil, err
	}
	return t.NewExtrinsics("XAssets.transfer", destAccountID, types.NewUCompactFromUInt(uint64(1)), types.NewUCompact(amountBigint))
}

func (t *Tx) NewThreshold(thresholdPublicKey, destAddress, aggSignature, aggPublicKey, controlBlock, message, scriptHash, transferAmount string, blockNumber int32) (*Transaction, error) {
	thresholdPublicKeyByte, err := types.HexDecodeString(thresholdPublicKey)
	if err != nil {
		return nil, err
	}

	destPublicKey, err := DecodeAddressToPublicKey(destAddress)
	if err != nil {
		return nil, err
	}
	destPublicKeyByte, err := types.HexDecodeString(destPublicKey)
	if err != nil {
		return nil, err
	}

	aggSignatureByte, err := types.HexDecodeString(aggSignature)
	if err != nil {
		return nil, err
	}

	aggPublicKeyByte, err := types.HexDecodeString(aggPublicKey)
	if err != nil {
		return nil, err
	}

	controlBlockByte, err := types.HexDecodeString(controlBlock)
	if err != nil {
		return nil, err
	}

	messageByte, err := types.HexDecodeString(message)
	if err != nil {
		return nil, err
	}

	scriptHashByte, err := types.HexDecodeString(scriptHash)
	if err != nil {
		return nil, err
	}

	amountBig, ok := new(big.Int).SetString(transferAmount, 10)
	if !ok {
		return nil, ErrNumber
	}

	passScriptCall, err := types.NewCall(t.metadata, "ThresholdSignature.pass_script", types.NewAccountID(thresholdPublicKeyByte), types.NewBytes(aggSignatureByte), types.NewBytes(aggPublicKeyByte), types.NewBytes(controlBlockByte), types.NewBytes(messageByte), types.NewBytes(scriptHashByte))
	if err != nil {
		return nil, err
	}

	execScriptCall, err := types.NewCall(t.metadata, "ThresholdSignature.exec_script", types.NewAccountID(destPublicKeyByte), types.NewU8(0), types.NewU128(*amountBig), types.NewU32(uint32(blockNumber)), types.NewU32(uint32(blockNumber))+types.NewU32(1000))
	if err != nil {
		return nil, err
	}

	arg := []types.Call{passScriptCall, execScriptCall}

	return t.NewExtrinsics("Utility.batch_all", arg)
}

func GenerateTransferSignData(rpcurl, scanUrl, from, to, amount string) ([]byte, error) {
	chain, err := NewChainWithRpc(rpcurl, scanUrl)
	if err != nil {
		return nil, err
	}
	metadata, err := chain.GetMetadataString()
	if err != nil {
		return nil, err
	}
	tx, err := NewTx(metadata)
	if err != nil {
		return nil, err
	}
	transaction, err := tx.NewBalanceTransferTx(to, amount)
	if err != nil {
		return nil, err
	}
	return chain.GetSignData(transaction, from)
}
