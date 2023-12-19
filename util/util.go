package util

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
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

func CreateFundStellarAccounts(pairs []*keypair.Full, count int, initialBalance string) error {

	masterClient := env.NewHorizonMasterClient()
	masterHzClient := masterClient.GetMaster()
	sourceKey := masterClient.GetSourceKey()

	hzClient := env.NewHorizonClient()

	ops := make([]txnbuild.Operation, count)

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
		pair := pairs[i]

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

	return nil
}

func InitTokenContract(kp *keypair.Full, contractIDAddress xdr.ScAddress) error {

	stellarClient := env.NewStellarClient(kp)
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
	_, err = stellarClient.InvokeAndProcessHostFunction("initialize", initArgs, contractIDAddress, kp)
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

func MintToken(kp *keypair.Full, contractAddr xdr.ScAddress, amount uint64, recipientAddr xdr.ScAddress) error { //stellarCl *env.StellarClient,
	stellarClient := env.NewStellarClient(kp)

	amountTo128Xdr := xdr.Int128Parts{Hi: 0, Lo: xdr.Uint64(amount)}

	amountSc, err := xdr.NewScVal(xdr.ScValTypeScvI128, amountTo128Xdr)
	if err != nil {
		panic(err)
	}
	mintTokenArgs, err := env.BuildMintTokenArgs(recipientAddr, amountSc)
	if err != nil {
		panic(err)
	}
	_, err = stellarClient.InvokeAndProcessHostFunction("mint", mintTokenArgs, contractAddr, kp)
	if err != nil {
		panic(err)
	}
	return nil
}

func GetTokenBalance(kp *keypair.Full, contractAddr xdr.ScAddress, balanceOf xdr.ScAddress) (uint64, error) { //xdr.TransactionMeta
	stellarClient := env.NewStellarClient(kp)

	GetTokenBalanceArgs, err := env.BuildGetTokenBalanceArgs(balanceOf)
	if err != nil {
		panic(err)
	}
	txMeta, err := stellarClient.InvokeAndProcessHostFunction("balance", GetTokenBalanceArgs, contractAddr, kp)
	if err != nil {
		panic(err)
	}

	bal := txMeta.V3.SorobanMeta.ReturnValue.I128

	if bal.Hi != 0 {
		return 0, errors.New("balance too large - cannot be mapped to uint64")
	} else {
		Uint64Bal := uint64(bal.Lo)
		return Uint64Bal, nil
	}

}

func MakeAccountAddress(kp keypair.KP) (xdr.ScAddress, error) {
	accountId, err := xdr.AddressToAccountId(kp.Address())
	if err != nil {
		return xdr.ScAddress{}, err
	}
	return xdr.NewScAddress(xdr.ScAddressTypeScAddressTypeAccount, accountId)
}
