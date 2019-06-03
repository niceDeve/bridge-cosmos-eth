package relayer

// -----------------------------------------------------
//      Relayer
//
//      Initializes the relayer service, which parses,
//      encodes, and packages named events on an Ethereum
//      Smart Contract for validator's to sign and send
//      to the Cosmos bridge.
// -----------------------------------------------------

import (
	"context"
	"fmt"
	"log"

  amino "github.com/tendermint/go-amino"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/events"
	"github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/contract"
	"github.com/swishlabsco/cosmos-ethereum-bridge/cmd/ebrelayer/txs"
)

// -------------------------------------------------------------------------
// Starts an event listener on a specific network, contract, and event
// -------------------------------------------------------------------------

func InitRelayer(cdc *amino.Codec, chainId string, provider string,
								 contractAddress common.Address, eventSig string,
								 validator sdk.AccAddress) error {

	// Start client with infura ropsten provider
	client, err := SetupWebsocketEthClient(provider)
	if err != nil {
		fmt.Errorf("%s", err)
	}
	fmt.Printf("\nStart ethereum websocket with provider: %s", provider)

	// We need the contract address in bytes[] for the query
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
	}

	// We will check logs for new events
	logs := make(chan types.Log)

	// Subscribe to the web socket, filter by contract and event, write results to logs
	fmt.Printf("\nStarting subscription filter on address: %s", contractAddress.Hex())
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Errorf("%s", err)
	} else {
		fmt.Printf("\nSubscription filter initialized!\n")
	}

	// Load Peggy Contract's ABI
	contractABI := contract.LoadABI()

	for {
		select {
		// Handle any errors
		case err := <-sub.Err():
			log.Fatal(err)
		// vLog is raw event data
		case vLog := <-logs:
			fmt.Println("\nNew event:")
			fmt.Println("BlockHash: ", vLog.BlockHash.Hex())
			fmt.Println("BlockNumber: ", vLog.BlockNumber)
			fmt.Println("TxHash: ", vLog.TxHash.Hex())

			// Check if the event is a 'LogLock' event
			if vLog.Topics[0].Hex() == eventSig {

				// Parse the event data into a new LockEvent using the contract's ABI
				event := events.NewLockEvent(contractABI, "LogLock", vLog.Data)

				// Parse the event's payload into a struct
				claim, claimErr := txs.ParsePayload(validator, &event)
				if claimErr != nil {
					log.Fatal(claimErr)
				}

				fmt.Printf("\nClaim information:\n%+v\n", claim)

				// Initiate the relay
			  relayErr := txs.RelayEvent(chainId, cdc, &claim)
			  if relayErr != nil {
					log.Fatal(relayErr)
				}
			}
		}
	}
	return fmt.Errorf("Error: Relayer timed out.")
}
