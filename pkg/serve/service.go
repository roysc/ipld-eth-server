// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package serve

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	ethnode "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/jmoiron/sqlx"

	"github.com/cerc-io/ipld-eth-server/v4/pkg/debug"
	"github.com/cerc-io/ipld-eth-server/v4/pkg/eth"
	"github.com/cerc-io/ipld-eth-server/v4/pkg/net"
)

const (
	PayloadChanBufferSize = 2000
)

// Server is the top level interface for streaming, converting to IPLDs, publishing,
// and indexing all chain data; screening this data; and serving it up to subscribed clients
// This service is compatible with the Ethereum service interface (node.Service)
type Server interface {
	// Start() and Stop()
	ethnode.Lifecycle
	APIs() []rpc.API
	Protocols() []p2p.Protocol
	// Pub-Sub handling event loop
	Serve(wg *sync.WaitGroup, screenAndServePayload <-chan eth.ConvertedPayload)
	// Method to subscribe to the service
	Subscribe(id rpc.ID, sub chan<- SubscriptionPayload, quitChan chan<- bool, params eth.SubscriptionSettings)
	// Method to unsubscribe from the service
	Unsubscribe(id rpc.ID)
	// Backend exposes the server's backend
	Backend() *eth.Backend
}

// Service is the underlying struct for the watcher
type Service struct {
	// Used to sync access to the Subscriptions
	sync.Mutex
	// Interface for filtering and serving data according to subscribed clients according to their specification
	Filterer eth.Filterer
	// Interface for fetching IPLD objects from IPFS
	IPLDFetcher eth.Fetcher
	// Interface for searching and retrieving CIDs from Postgres index
	Retriever eth.Retriever
	// Used to signal shutdown of the service
	QuitChan chan bool
	// A mapping of rpc.IDs to their subscription channels, mapped to their subscription type (hash of the StreamFilters)
	Subscriptions map[common.Hash]map[rpc.ID]Subscription
	// A mapping of subscription params hash to the corresponding subscription params
	SubscriptionTypes map[common.Hash]eth.SubscriptionSettings
	// Underlying db
	db *sqlx.DB
	// wg for syncing serve processes
	serveWg *sync.WaitGroup
	// rpc client for forwarding cache misses
	client *rpc.Client
	// whether the proxied client supports state diffing
	supportsStateDiffing bool
	// timeout for statediff RPC calls
	stateDiffTimeout time.Duration
	// backend for the server
	backend *eth.Backend
	// whether to forward eth_calls directly to proxy node
	forwardEthCalls bool
	// whether to forward eth_getStorageAt directly to proxy node
	forwardGetStorageAt bool
	// whether to forward all calls to proxy node if they throw an error locally
	proxyOnError bool
	// eth node network id
	nodeNetworkId string
}

// NewServer creates a new Server using an underlying Service struct
func NewServer(settings *Config) (Server, error) {
	sap := new(Service)
	sap.Retriever = eth.NewCIDRetriever(settings.DB)
	sap.IPLDFetcher = eth.NewIPLDFetcher(settings.DB)
	sap.Filterer = eth.NewResponseFilterer()
	sap.db = settings.DB
	sap.QuitChan = make(chan bool)
	sap.Subscriptions = make(map[common.Hash]map[rpc.ID]Subscription)
	sap.SubscriptionTypes = make(map[common.Hash]eth.SubscriptionSettings)
	sap.client = settings.Client
	sap.supportsStateDiffing = settings.SupportStateDiff
	sap.stateDiffTimeout = settings.StateDiffTimeout
	sap.forwardEthCalls = settings.ForwardEthCalls
	sap.forwardGetStorageAt = settings.ForwardGetStorageAt
	sap.proxyOnError = settings.ProxyOnError
	sap.nodeNetworkId = settings.NodeNetworkID
	var err error
	sap.backend, err = eth.NewEthBackend(sap.db, &eth.Config{
		ChainConfig:      settings.ChainConfig,
		VMConfig:         vm.Config{NoBaseFee: true},
		DefaultSender:    settings.DefaultSender,
		RPCGasCap:        settings.RPCGasCap,
		GroupCacheConfig: settings.GroupCache,
	})
	return sap, err
}

// Protocols exports the services p2p protocols, this service has none
func (sap *Service) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

// APIs returns the RPC descriptors the watcher service offers
func (sap *Service) APIs() []rpc.API {
	networkID, _ := strconv.ParseUint(sap.nodeNetworkId, 10, 64)
	apis := []rpc.API{
		{
			Namespace: APIName,
			Version:   APIVersion,
			Service:   NewPublicServerAPI(sap, sap.client),
			Public:    true,
		},
		{
			Namespace: net.APIName,
			Version:   net.APIVersion,
			Service:   net.NewPublicNetAPI(networkID, sap.client),
			Public:    true,
		},
	}
	conf := eth.APIConfig{
		SupportsStateDiff:   sap.supportsStateDiffing,
		ForwardEthCalls:     sap.forwardEthCalls,
		ForwardGetStorageAt: sap.forwardGetStorageAt,
		ProxyOnError:        sap.proxyOnError,
		StateDiffTimeout:    sap.stateDiffTimeout,
	}
	ethAPI, err := eth.NewPublicEthAPI(sap.backend, sap.client, conf)
	if err != nil {
		log.Fatalf("unable to create public eth api: %v", err)
	}

	debugTracerAPI := tracers.APIs(&debug.Backend{Backend: *sap.backend})[0]

	return append(apis,
		rpc.API{
			Namespace: eth.APIName,
			Version:   eth.APIVersion,
			Service:   ethAPI,
			Public:    true,
		},
		debugTracerAPI,
	)
}

// Serve listens for incoming converter data off the screenAndServePayload from the Sync process
// It filters and sends this data to any subscribers to the service
// This process can also be stood up alone, without an screenAndServePayload attached to a Sync process
// and it will hang on the WaitGroup indefinitely, allowing the Service to serve historical data requests only
func (sap *Service) Serve(wg *sync.WaitGroup, screenAndServePayload <-chan eth.ConvertedPayload) {
	sap.serveWg = wg
	go func() {
		wg.Add(1)
		defer wg.Done()
		for {
			select {
			case payload := <-screenAndServePayload:
				sap.filterAndServe(payload)
			case <-sap.QuitChan:
				log.Info("quiting eth ipld server process")
				return
			}
		}
	}()
	log.Info("eth ipld server process successfully spun up")
}

// filterAndServe filters the payload according to each subscription type and sends to the subscriptions
func (sap *Service) filterAndServe(payload eth.ConvertedPayload) {
	log.Debug("sending eth ipld payload to subscriptions")
	sap.Lock()
	sap.serveWg.Add(1)
	defer sap.Unlock()
	defer sap.serveWg.Done()
	for ty, subs := range sap.Subscriptions {
		// Retrieve the subscription parameters for this subscription type
		subConfig, ok := sap.SubscriptionTypes[ty]
		if !ok {
			log.Errorf("eth ipld server subscription configuration for subscription type %s not available", ty.Hex())
			sap.closeType(ty)
			continue
		}
		if subConfig.End.Int64() > 0 && subConfig.End.Int64() < payload.Block.Number().Int64() {
			// We are not out of range for this subscription type
			// close it, and continue to the next
			sap.closeType(ty)
			continue
		}
		response, err := sap.Filterer.Filter(subConfig, payload)
		if err != nil {
			log.Errorf("eth ipld server filtering error: %v", err)
			sap.closeType(ty)
			continue
		}
		responseRLP, err := rlp.EncodeToBytes(response)
		if err != nil {
			log.Errorf("eth ipld server rlp encoding error: %v", err)
			continue
		}
		for id, sub := range subs {
			select {
			case sub.PayloadChan <- SubscriptionPayload{Data: responseRLP, Err: "", Flag: EmptyFlag, Height: response.BlockNumber.Int64()}:
				log.Debugf("sending eth ipld server payload to subscription %s", id)
			default:
				log.Infof("unable to send eth ipld payload to subscription %s; channel has no receiver", id)
			}
		}
	}
}

// Subscribe is used by the API to remotely subscribe to the service loop
// The params must be rlp serializable and satisfy the SubscriptionSettings() interface
func (sap *Service) Subscribe(id rpc.ID, sub chan<- SubscriptionPayload, quitChan chan<- bool, params eth.SubscriptionSettings) {
	sap.serveWg.Add(1)
	defer sap.serveWg.Done()
	log.Infof("new eth ipld subscription %s", id)
	subscription := Subscription{
		ID:          id,
		PayloadChan: sub,
		QuitChan:    quitChan,
	}
	// Subscription type is defined as the hash of the rlp-serialized subscription settings
	by, err := rlp.EncodeToBytes(params)
	if err != nil {
		sendNonBlockingErr(subscription, err)
		sendNonBlockingQuit(subscription)
		return
	}
	subscriptionType := crypto.Keccak256Hash(by)
	if !params.BackFillOnly {
		// Add subscriber
		sap.Lock()
		if sap.Subscriptions[subscriptionType] == nil {
			sap.Subscriptions[subscriptionType] = make(map[rpc.ID]Subscription)
		}
		sap.Subscriptions[subscriptionType][id] = subscription
		sap.SubscriptionTypes[subscriptionType] = params
		sap.Unlock()
	}
	// If the subscription requests a backfill, use the Postgres index to lookup and retrieve historical data
	// Otherwise we only filter new data as it is streamed in from the state diffing geth node
	if params.BackFill || params.BackFillOnly {
		if err := sap.sendHistoricalData(subscription, id, params); err != nil {
			sendNonBlockingErr(subscription, fmt.Errorf("eth ipld server subscription backfill error: %v", err))
			sendNonBlockingQuit(subscription)
			return
		}
	}
}

// sendHistoricalData sends historical data to the requesting subscription
func (sap *Service) sendHistoricalData(sub Subscription, id rpc.ID, params eth.SubscriptionSettings) error {
	log.Infof("sending eth ipld historical data to subscription %s", id)
	// Retrieve cached CIDs relevant to this subscriber
	var endingBlock int64
	var startingBlock int64
	var err error
	startingBlock, err = sap.Retriever.RetrieveFirstBlockNumber()
	if err != nil {
		return err
	}
	if startingBlock < params.Start.Int64() {
		startingBlock = params.Start.Int64()
	}
	endingBlock, err = sap.Retriever.RetrieveLastBlockNumber()
	if err != nil {
		return err
	}
	if endingBlock > params.End.Int64() && params.End.Int64() > 0 && params.End.Int64() > startingBlock {
		endingBlock = params.End.Int64()
	}
	log.Debugf("eth ipld historical data starting block: %d", params.Start.Int64())
	log.Debugf("eth ipld historical data ending block: %d", endingBlock)
	go func() {
		sap.serveWg.Add(1)
		defer sap.serveWg.Done()
		for i := startingBlock; i <= endingBlock; i++ {
			select {
			case <-sap.QuitChan:
				log.Infof("ethereum historical data feed to subscription %s closed", id)
				return
			default:
			}
			cidWrappers, empty, err := sap.Retriever.Retrieve(params, i)
			if err != nil {
				sendNonBlockingErr(sub, fmt.Errorf("eth ipld server cid retrieval error at block %d\r%s", i, err.Error()))
				continue
			}
			if empty {
				continue
			}
			for _, cids := range cidWrappers {
				response, err := sap.IPLDFetcher.Fetch(cids)
				if err != nil {
					sendNonBlockingErr(sub, fmt.Errorf("eth ipld server ipld fetching error at block %d\r%s", i, err.Error()))
					continue
				}
				responseRLP, err := rlp.EncodeToBytes(response)
				if err != nil {
					log.Error(err)
					continue
				}
				select {
				case sub.PayloadChan <- SubscriptionPayload{Data: responseRLP, Err: "", Flag: EmptyFlag, Height: response.BlockNumber.Int64()}:
					log.Debugf("eth ipld server sending historical data payload to subscription %s", id)
				default:
					log.Infof("eth ipld server unable to send backFill payload to subscription %s; channel has no receiver", id)
				}
			}
		}
		// when we are done backfilling send an empty payload signifying so in the msg
		select {
		case sub.PayloadChan <- SubscriptionPayload{Data: nil, Err: "", Flag: BackFillCompleteFlag}:
			log.Debugf("eth ipld server sending backFill completion notice to subscription %s", id)
		default:
			log.Infof("eth ipld server unable to send backFill completion notice to subscription %s", id)
		}
	}()
	return nil
}

// Unsubscribe is used by the API to remotely unsubscribe to the StateDiffingService loop
func (sap *Service) Unsubscribe(id rpc.ID) {
	log.Infof("unsubscribing %s from the eth ipld server", id)
	sap.Lock()
	for ty := range sap.Subscriptions {
		delete(sap.Subscriptions[ty], id)
		if len(sap.Subscriptions[ty]) == 0 {
			// If we removed the last subscription of this type, remove the subscription type outright
			delete(sap.Subscriptions, ty)
			delete(sap.SubscriptionTypes, ty)
		}
	}
	sap.Unlock()
}

// Start is used to begin the service
// This is mostly just to satisfy the node.Service interface
func (sap *Service) Start() error {
	log.Info("starting eth ipld server")
	wg := new(sync.WaitGroup)
	payloadChan := make(chan eth.ConvertedPayload, PayloadChanBufferSize)
	sap.Serve(wg, payloadChan)
	return nil
}

// Stop is used to close down the service
// This is mostly just to satisfy the node.Service interface
func (sap *Service) Stop() error {
	log.Infof("stopping eth ipld server")
	sap.Lock()
	close(sap.QuitChan)
	sap.close()
	sap.Unlock()
	return nil
}

// Backend exposes the server's backend
func (sap *Service) Backend() *eth.Backend {
	return sap.backend
}

// close is used to close all listening subscriptions
// close needs to be called with subscription access locked
func (sap *Service) close() {
	log.Infof("closing all eth ipld server subscriptions")
	for subType, subs := range sap.Subscriptions {
		for _, sub := range subs {
			sendNonBlockingQuit(sub)
		}
		delete(sap.Subscriptions, subType)
		delete(sap.SubscriptionTypes, subType)
	}
}

// closeType is used to close all subscriptions of given type
// closeType needs to be called with subscription access locked
func (sap *Service) closeType(subType common.Hash) {
	log.Infof("closing all eth ipld server subscriptions of type %s", subType.String())
	subs := sap.Subscriptions[subType]
	for _, sub := range subs {
		sendNonBlockingQuit(sub)
	}
	delete(sap.Subscriptions, subType)
	delete(sap.SubscriptionTypes, subType)
}
