package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	_ "github.com/filecoin-project/lotus/lib/sigs/bls"
	"github.com/urfave/cli/v2"
)

/**
 * This program imports and address, creates and exports a new address with a private key and allocates some
 * a small amount of FIL to it
 */

func main() {

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "type",
			Aliases: []string{"t"},
			Value:   "bls",
			Usage:   "specify key type to generate (bls or secp256k1)",
		},
	}
	app.Action = func(cctx *cli.Context) error {
		ctx := context.Background()

		memks := wallet.NewMemKeyStore()
		w, err := wallet.NewWallet(memks)
		if err != nil {
			return err
		}

		var kt crypto.SigType
		switch cctx.String("type") {
		case "bls":
			kt = crypto.SigTypeBLS
		case "secp256k1":
			kt = crypto.SigTypeSecp256k1
		default:
			return fmt.Errorf("unrecognized key type: %q", cctx.String("type"))
		}

		kaddr, err := w.WalletNew(cctx.Context, kt)
		if err != nil {
			return err
		}

		ki, err := w.WalletExport(cctx.Context, kaddr)
		if err != nil {
			return err
		}

		fi, err := os.Create("burgers.key")
		if err != nil {
			return err
		}
		b, err := json.Marshal(ki)
		if err != nil {
			return err
		}

		if _, err := fi.Write(b); err != nil {
			return fmt.Errorf("failed to write key info to file: %w", err)
		}

		fmt.Println("Generated new key: ", kaddr)

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("Unable to get current directory: %v", err)
		}
		fdata, err := ioutil.ReadFile(path.Join(wd, "client.private"))
		if err != nil {
			return fmt.Errorf("Unable to import private key file: %v", err)
		}
		var iki types.KeyInfo
		data, err := hex.DecodeString(strings.TrimSpace(string(fdata)))
		if err != nil {
			return fmt.Errorf("Unable to decode hex string: %v", err)
		}
		if err := json.Unmarshal(data, &iki); err != nil {
			return fmt.Errorf("Unable to unmarshal keyinfo: %v", err)
		}
		addr, err := w.WalletImport(ctx, &iki)

		fmt.Println("Imported new key: ", addr)

		api, closer, err := client.NewFullNodeRPC(ctx, "ws://35.184.58.104:8080/rpc/v0", http.Header{
			// This token can write msgs to mempool but not sign them
			"Authorization": []string{fmt.Sprintf("Bearer %s", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiXX0.K7gSaQ4WchdDktdsC0yiLTPKL1fwxTAciLgEO6zuW8g")},
		})
		defer func() {
			closer()
			err2 := fi.Close()
			if err == nil {
				err = err2
			}
		}()

		val, err := types.ParseFIL("0.01")

		method := abi.MethodNum(uint64(0))
		msg := &types.Message{
			From:   addr,
			To:     kaddr,
			Value:  types.BigInt(val),
			Method: method,
		}

		msg, err = api.GasEstimateMessageGas(ctx, msg, nil, types.EmptyTSK)
		// TODO save nonce from local data store
		msg.Nonce, err = api.MpoolGetNonce(ctx, msg.From)
		if err != nil {
			return err
		}

		mbl, err := msg.ToStorageBlock()
		if err != nil {
			return err
		}

		sig, err := w.WalletSign(ctx, msg.From, mbl.Cid().Bytes(), lapi.MsgMeta{})
		if err != nil {
			return err
		}

		smsg := &types.SignedMessage{
			Message:   *msg,
			Signature: *sig,
		}

		fmt.Println("Signed message: ", smsg)

		if _, err := api.MpoolPush(ctx, smsg); err != nil {
			return fmt.Errorf("MpoolPush failed with error: %v", err)
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
