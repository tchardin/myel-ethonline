package main

import (
    "context"
    "crypto/ecdsa"
    "fmt"
    "log"
    "math/big"
    "time"

    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/crypto"

    channelverify "github.com/tchardin/myel-ethonline/oracle/go-client/contracts"

)

func main() {
    client, err := ethclient.Dial("http://localhost:7545")
    if err != nil {
        log.Fatal(err)
    }

    // Get private key from Ganache and insert here
    privateKey, err := crypto.HexToECDSA("447f92f685cc09d544c5b0dc83cc8358a5ea3b7740ad2a36ed8215f29c86ffab")
    if err != nil {
        log.Fatal(err)
    }

    publicKey := privateKey.Public()
    publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
    if !ok {
        log.Fatal("error casting public key to ECDSA")
    }

    fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
    nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
    if err != nil {
        log.Fatal(err)
    }

    gasPrice, err := client.SuggestGasPrice(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    auth := bind.NewKeyedTransactor(privateKey)
    auth.Nonce = big.NewInt(int64(nonce))
    auth.Value = big.NewInt(0)     // in wei
    auth.GasLimit = uint64(300000) // in units
    auth.GasPrice = gasPrice


    // Insert smart contract address
    address := common.HexToAddress("0xa0cEeCf0B1f9Ed7dF6287A09d188E949839DECC2")
    instance, err := channelverify.NewChannelverify(address, client)
    if err != nil {
        log.Fatal(err)
    }

    // Insert oracle contract address
    _oracle := common.HexToAddress("0x8af9fddd9f09e7b48c4291cf8d1ef5fe6ef1d806")
    // Insert Chainlink job id
    _jobId := "2263df3fc8524eb28e8eb28266c152db"
    // Insert channel address
    _channel := "f033012"

    fmt.Println("Verifying Payment Channel with Lotus Node ...")
    tx, err := instance.VerifyChannel(auth, _oracle, _jobId, _channel)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("tx sent: %s", tx.Hash().Hex()) // tx sent: 0x8d490e535678e9a24360e955d75b27ad307bdfb97a1dca51d0f3035dcee3e870


}
