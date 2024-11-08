package main

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type DistributeRequest struct {
	Address string `json:"address"`
}

type DistributeResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func distributeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the query parameters
	query := r.URL.Query()
	address := query.Get("address")

	// Validate the address
	if address == "" || !strings.HasPrefix(address, "0x") {
		http.Error(w, "Invalid address format", http.StatusBadRequest)
		return
	}

	// Process the request
	err := processRequest(address)
	if err != nil {
		http.Error(w, "Error processing request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the response
	response := DistributeResponse{
		Message: "Distribution successful",
		Data:    nil, // Replace with any additional data you want to return
	}

	// Encode the response as JSON
	json.NewEncoder(w).Encode(response)
}

func processRequest(address string) error {
	// Replace with your Ethereum node's RPC endpoint
	client, err := ethclient.Dial("http://localhost:8545")
	if err != nil {
		return err
	}

	// Replace with your private key (in hex format)
	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		return err
	}

	// Replace with the amount to send (in wei)
	amount := big.NewInt(1000000000000000000) // 1 ETH

	// Get the sender's address
	sender := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create a transaction
	nonce, err := client.NonceAt(context.Background(), sender, nil)
	if err != nil {
		return err
	}

	tx := types.NewTransaction(nonce, common.HexToAddress(address), amount, 21000, big.NewInt(875000000), nil)

	// Sign the transaction
	signer := types.FrontierSigner{}
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return err
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	// Transaction sent successfully
	return nil
}

func main() {
	// Replace "0.0.0.0:8080" with your desired address and port
	http.HandleFunc("/distribute", distributeHandler)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
