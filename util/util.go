package util

import (
	"crypto/rand"
	"encoding/binary"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"log"
	mathrand "math/rand"
	"perun.network/perun-stellar-backend/channel"
	"perun.network/perun-stellar-backend/channel/env"
	"perun.network/perun-stellar-backend/channel/types"
	"perun.network/perun-stellar-backend/wallet"
)

const NETWORK_PASSPHRASE = "Standalone Network ; February 2017"

type StellarClient struct {
	horizonClient *horizonclient.Client
}

func (s *StellarClient) SubmitTx(tx *txnbuild.Transaction) error {
	_, err := s.horizonClient.SubmitTransaction(tx)
	if err != nil {
		panic(err)
	}
	return err
}

func CreateSignedTransaction(signers []*keypair.Full, txParams txnbuild.TransactionParams,
) (*txnbuild.Transaction, error) {
	tx, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, err
	}

	for _, signer := range signers {
		tx, err = tx.Sign(NETWORK_PASSPHRASE, signer)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

func DeployStandard(hzClient *env.StellarClient, kp *keypair.Full, hz horizon.Account, contractPath string) (xdr.ScAddress, xdr.Hash) {
	// Install contract
	cl := hzClient.GetHorizonClient()

	installContractOpInstall := channel.AssembleInstallContractCodeOp(kp.Address(), contractPath)
	preFlightOp, minFeeInstall := env.PreflightHostFunctions(cl, &hz, *installContractOpInstall)
	txParamsInstall := env.GetBaseTransactionParamsWithFee(&hz, int64(minFeeInstall), &preFlightOp)
	txSignedInstall, err := CreateSignedTransaction([]*keypair.Full{kp}, txParamsInstall)
	if err != nil {
		panic(err)
	}
	_, err = cl.SubmitTransaction(txSignedInstall)
	if err != nil {
		panic(err)
	}

	// Create the contract
	createContractOp := channel.AssembleCreateContractOp(kp.Address(), contractPath, "a1", NETWORK_PASSPHRASE)
	preFlightOpCreate, minFeeCreate := env.PreflightHostFunctions(cl, &hz, *createContractOp)
	txParamsCreate := env.GetBaseTransactionParamsWithFee(&hz, int64(minFeeCreate), &preFlightOpCreate)
	txSignedCreate, err := CreateSignedTransaction([]*keypair.Full{kp}, txParamsCreate)
	if err != nil {
		panic(err)
	}
	_, err = cl.SubmitTransaction(txSignedCreate)
	if err != nil {
		panic(err)
	}
	contractID := preFlightOpCreate.Ext.SorobanData.Resources.Footprint.ReadWrite[0].MustContractData().Contract.ContractId
	contractHash := preFlightOpCreate.Ext.SorobanData.Resources.Footprint.ReadOnly[0].MustContractCode().Hash
	contractIDAddress := xdr.ScAddress{
		Type:       xdr.ScAddressTypeScAddressTypeContract,
		ContractId: contractID,
	}
	return contractIDAddress, contractHash

}

func CreateFundNewRandomStellarKP(count int, initialBalance string) ([]*keypair.Full, []txnbuild.Account) {

	masterClient := env.NewHorizonMasterClient()
	masterHzClient := masterClient.GetMaster()
	sourceKey := masterClient.GetSourceKey()

	hzClient := env.NewHorizonClient()

	pairs := make([]*keypair.Full, count)
	ops := make([]txnbuild.Operation, count)

	// kp := keypair.MustRandom()
	accReq := horizonclient.AccountRequest{AccountID: sourceKey.Address()}
	sourceAccount, err := masterHzClient.AccountDetail(accReq)
	if err != nil {
		panic(err)
	}

	masterAccount := txnbuild.SimpleAccount{
		AccountID: sourceKey.Address(),
		Sequence:  sourceAccount.Sequence,
	}

	// use masteraccount to generate new accounts
	for i := 0; i < count; i++ {
		pair, _ := keypair.Random()
		pairs[i] = pair

		ops[i] = &txnbuild.CreateAccount{
			SourceAccount: masterAccount.AccountID,
			Destination:   pair.Address(),
			Amount:        initialBalance,
		}
	}

	txParams := env.GetBaseTransactionParamsWithFee(&masterAccount, txnbuild.MinBaseFee, ops...)

	txSigned, err := env.CreateSignedTransactionWithParams([]*keypair.Full{sourceKey}, txParams)

	if err != nil {
		panic(err)
	}
	_, err = hzClient.SubmitTransaction(txSigned)
	if err != nil {
		panic(err)
	}

	accounts := make([]txnbuild.Account, count)
	for i, kp := range pairs {
		request := horizonclient.AccountRequest{AccountID: kp.Address()}
		account, err := hzClient.AccountDetail(request)
		if err != nil {
			panic(err)
		}

		accounts[i] = &account
	}

	for _, keys := range pairs {
		log.Printf("Funded %s (%s) with %s XLM.\n",
			keys.Seed(), keys.Address(), initialBalance)
	}

	return pairs, accounts
}

func InitTokenContract(kp *keypair.Full, contractIDAddress xdr.ScAddress) error {

	stellarClient := env.NewStellarClient(kp)
	// cl := stellarClient.GetHorizonClient()
	adminScAddr, err := types.MakeAccountAddress(kp)
	if err != nil {
		panic(err)
	}

	tokenParams := channel.NewTokenParams()
	decimals := tokenParams.GetDecimals()
	name := tokenParams.GetName()
	symbol := tokenParams.GetSymbol()

	initArgs, err := channel.BuildInitTokenArgs(adminScAddr, decimals, name, symbol)
	if err != nil {
		panic(err)
	}
	auth := []xdr.SorobanAuthorizationEntry{}
	_, err = stellarClient.InvokeAndProcessHostFunction("initialize", initArgs, contractIDAddress, kp, auth)
	if err != nil {
		panic(err)
	}

	return nil
}

func MakeRandPerunWallet() (*wallet.EphemeralWallet, *wallet.Account, *keypair.Full) {
	w := wallet.NewEphemeralWallet()

	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}

	seed := binary.LittleEndian.Uint64(b[:])

	r := mathrand.New(mathrand.NewSource(int64(seed)))

	acc, kp, err := w.AddNewAccount(r)
	if err != nil {
		panic(err)
	}
	return w, acc, kp
}

func NewAssetFromScAddress(contractAddr xdr.ScAddress) types.StellarAsset {
	contractAsset, err := types.NewStellarAssetFromScAddress(contractAddr)
	if err != nil {
		panic(err)
	}
	return *contractAsset
}

func i128Param(hi int64, lo uint64) xdr.ScVal {
	i128 := &xdr.Int128Parts{
		Hi: xdr.Int64(hi),
		Lo: xdr.Uint64(lo),
	}
	return xdr.ScVal{
		Type: xdr.ScValTypeScvI128,
		I128: i128,
	}
}

func MintToken(kp *keypair.Full, contractAddr xdr.ScAddress, amount int64, recipientAddr xdr.ScAddress) error { //stellarCl *env.StellarClient,
	stellarClient := env.NewStellarClient(kp)
	// cl := stellarClient.GetHorizonClient()

	// amount128 := i128Param(0, amount)

	amountSc, err := xdr.NewScVal(xdr.ScValTypeScvI64, xdr.Int64(amount))
	if err != nil {
		panic(err)
	}
	mintTokenArgs, err := env.BuildMintTokenArgs(recipientAddr, amountSc)
	if err != nil {
		panic(err)
	}
	// contractAsset := NewAssetFromScAddress(contractAddr)
	_, err = stellarClient.InvokeAndProcessHostFunction("mint", mintTokenArgs, contractAddr, kp, []xdr.SorobanAuthorizationEntry{})
	if err != nil {
		panic(err)
	}
	return nil
}

func MakeAccountAddress(kp keypair.KP) (xdr.ScAddress, error) {
	accountId, err := xdr.AddressToAccountId(kp.Address())
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountId)
}
