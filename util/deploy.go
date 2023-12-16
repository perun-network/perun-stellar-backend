package util

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
	"perun.network/perun-stellar-backend/channel"

	"perun.network/perun-stellar-backend/channel/env"
)

func Deploy(kp *keypair.Full, contractPath string) (xdr.ScAddress, xdr.Hash) {
	// Install contract
	deployerClient := env.NewStellarClient(kp) //.GetHorizonClient()
	hzClient := deployerClient.GetHorizonClient()
	deployerAccReq := horizonclient.AccountRequest{AccountID: kp.Address()}
	deployerAcc, err := hzClient.AccountDetail(deployerAccReq)
	if err != nil {
		panic(err)
	}

	installContractOpInstall := channel.AssembleInstallContractCodeOp(kp.Address(), contractPath)
	preFlightOp, minFeeInstall := env.PreflightHostFunctions(hzClient, &deployerAcc, *installContractOpInstall)
	txParamsInstall := env.GetBaseTransactionParamsWithFee(&deployerAcc, int64(minFeeInstall), &preFlightOp)
	txSignedInstall, err := CreateSignedTransaction([]*keypair.Full{kp}, txParamsInstall)
	if err != nil {
		panic(err)
	}
	_, err = hzClient.SubmitTransaction(txSignedInstall)
	if err != nil {
		panic(err)
	}

	// Create the contract
	createContractOp := channel.AssembleCreateContractOp(kp.Address(), contractPath, "a1", NETWORK_PASSPHRASE)
	preFlightOpCreate, minFeeCreate := env.PreflightHostFunctions(hzClient, &deployerAcc, *createContractOp)
	txParamsCreate := env.GetBaseTransactionParamsWithFee(&deployerAcc, int64(minFeeCreate), &preFlightOpCreate)
	txSignedCreate, err := CreateSignedTransaction([]*keypair.Full{kp}, txParamsCreate)
	if err != nil {
		panic(err)
	}
	_, err = hzClient.SubmitTransaction(txSignedCreate)
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

// func MintAsset() {
// 	env.BuildMint

// }
