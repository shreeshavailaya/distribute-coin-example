package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
)

var (
	client           *ethclient.Client // Shared Ethereum client connection
	transactionQueue struct {
		mutex        sync.Mutex
		Transactions []TransactionRequest
	}
	transactionStatusMap = make(map[string]*TransactionStatus)
	currentNonce         uint64 //for future use
	currentNonceMutex    sync.Mutex
	clientMutex          sync.Mutex
	ethereumRPCURL       = "http://localhost:8545"
	transactionChannel   = make(chan TransactionRequest, 100) // Buffered channel for transaction requests
)

type TransactionRequest struct {
	UUID    string   `json:"uuid"`
	Address string   `json:"address"`
	Amount  *big.Int `json:"amount"`
}

type TransactionStatus struct {
	UUID      string `json:"uuid"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

type DistributeRequest struct {
	Address string `json:"address"`
}

type DistributeResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func initializeEthereumClient() error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	// Reconnect if the client is not already initialized
	if client != nil {
		return nil // Already initialized
	}

	var err error
	client, err = ethclient.Dial(ethereumRPCURL)
	if err != nil {
		log.Printf("Failed to connect to Ethereum client: %v", err)
	}
	return err
}

func checkClientHealth() error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if client == nil {
		log.Printf("client is not initialized")
		return errors.New("client is not initialized")
	}

	// Try to fetch the latest block number to confirm the connection is healthy
	_, err := client.BlockNumber(context.Background())
	return err
}

func distributeHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the query parameters
	query := r.URL.Query()
	address := query.Get("address")

	// Replace with the amount to send (in wei)
	amount := big.NewInt(1000000000000000000) // 1 ETH

	// Validate the address
	if address == "" || !strings.HasPrefix(address, "0x") {
		http.Error(w, "Invalid address format", http.StatusBadRequest)
		return
	}

	// Add the request to the transaction queue
	reqUUID := uuid.New().String()
	transactionQueue.mutex.Lock()
	transactionQueue.Transactions = append(transactionQueue.Transactions, TransactionRequest{UUID: reqUUID, Address: address, Amount: amount})
	transactionQueue.mutex.Unlock()

	// Respond with the UUID for tracking
	response := DistributeResponse{
		Message: "Request received, processing transaction",
		Data:    map[string]interface{}{"uuid": reqUUID},
	}

	log.Println("uuid: ", reqUUID)

	// Encode the response as JSON
	json.NewEncoder(w).Encode(response)

	// Send the transaction request to the channel for processing
	transactionChannel <- TransactionRequest{UUID: reqUUID, Address: address, Amount: amount}

}

/*
Todo: check if we can add the logic of generating the nonce in the application and
prevent redundant networkcalls if the transaction is initiated only from this service.
*/
func getNonce(client *ethclient.Client, sender common.Address) (uint64, error) {
	// Lock the mutex to safely update currentNonce
	currentNonceMutex.Lock()
	defer currentNonceMutex.Unlock()

	// Sync the local nonce with the Ethereum network's nonce
	nonce, err := client.NonceAt(context.Background(), sender, nil)
	if err != nil {
		return 0, err
	}

	return nonce, nil
}

func updateTransactionStatus(uuid string, status string, errorMessage string) {
	transactionStatusMap[uuid] = &TransactionStatus{
		UUID:      uuid,
		Status:    status,
		Error:     errorMessage,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func processRequest(req TransactionRequest) {

	//check if there is valid connection, else try to reconnect
	err := checkClientHealth()
	if err != nil {
		err := initializeEthereumClient()
		if err != nil {
			updateTransactionStatus(req.UUID, "failed", "Error initializing Ethereum client")
			return
		}
	}

	// Replace with your private key (in hex format)
	privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		log.Printf("Failed to decode private key: %v", err)
		updateTransactionStatus(req.UUID, "failed", "Error decoding private key")
		return
	}

	// Get the sender's address
	sender := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Retrieve the local nonce and increment it
	nonce, err := getNonce(client, sender)
	if err != nil {
		updateTransactionStatus(req.UUID, "failed", "Error getting nonce: "+err.Error())
		return
	}

	// Create a transaction
	tx := types.NewTransaction(nonce, common.HexToAddress(req.Address), req.Amount, 21000, big.NewInt(875000000), nil)

	// Sign the transaction
	signer := types.FrontierSigner{}
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		log.Printf("Failed to sign transaction: %v", err)
		updateTransactionStatus(req.UUID, "failed", "Error signing transaction")
		return
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Printf("Failed to send transaction: %v", err)
		updateTransactionStatus(req.UUID, "failed", "Error sending transaction")
		return
	}

	// Transaction sent successfully
	updateTransactionStatus(req.UUID, "success", "Transaction sent successfully")
}

func getTransactionStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the UUID parameter
	query := r.URL.Query()
	uuid := query.Get("uuid")
	if uuid == "" {
		http.Error(w, "UUID is required", http.StatusBadRequest)
		return
	}

	// Check if transaction status exists for the given UUID
	status, exists := transactionStatusMap[uuid]
	if !exists {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Respond with the transaction status
	json.NewEncoder(w).Encode(status)
}

func main() {

	// Initialize the Ethereum client
	if err := initializeEthereumClient(); err != nil {
		log.Fatalf("Failed to initialize Ethereum client: %v", err)
	}
	defer client.Close()

	// Replace "0.0.0.0:8080" with your desired address and port
	http.HandleFunc("/distribute", distributeHandler)

	/* get the uuid from the response(or logs)

	sample request:
	   curl -X GET "http://localhost:8080/transaction/status?uuid=31037daf-38da-4e9a-898f-4af9e8950061"

	sample output:

	    {"uuid":"31037daf-38da-4e9a-898f-4af9e8950061",
			 "status":"success",
	     "error":"Transaction sent successfully",
			 "timestamp":"2024-11-21T09:05:57+05:30"}
	*/
	http.HandleFunc("/transaction/status", getTransactionStatusHandler)

	// Start processing transaction requests asynchronously
	go func() {
		for req := range transactionChannel {
			processRequest(req)
		}
	}()
	// Start the HTTP server
	log.Println("Server listening on port 8080...")

	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
