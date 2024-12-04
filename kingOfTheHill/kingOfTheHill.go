package kingOfTheHill

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethanhosier/pumpfun-trade-bot/pumpfun"
)

const (
	bufferSize = 1024
)

type KingOfTheHillClient struct {
	pumpfunClient *pumpfun.PumpFunClient

	listeners   map[string]chan<- *pumpfun.CoinData
	listenersMu sync.Mutex
}

func NewKingOfTheHillClient(pumpfunClient *pumpfun.PumpFunClient) *KingOfTheHillClient {
	return &KingOfTheHillClient{
		pumpfunClient: pumpfunClient,
		listeners:     make(map[string]chan<- *pumpfun.CoinData),
	}
}

func (k *KingOfTheHillClient) KingOfTheHillCoinData() (*pumpfun.CoinData, error) {
	return k.pumpfunClient.KingOfTheHillCoinData()
}

func (k *KingOfTheHillClient) Start(minPollingInterval time.Duration, maxRetries int) error {
	numConsecutiveErrors := 0
	errChan := make(chan error)

	go func() {
		for {
			coinData, err := k.KingOfTheHillCoinData()
			if err != nil {
				log.Printf("Error fetching king of the hill coin data %d: %v", numConsecutiveErrors+1, err)
				numConsecutiveErrors++

				if numConsecutiveErrors > maxRetries {
					errChan <- fmt.Errorf("too many consecutive errors while fetching king of the hill coin data: %v", err)
					return
				}

				continue
			}

			// Reset consecutive errors on success
			numConsecutiveErrors = 0
			k.notifyListeners(coinData)
			time.Sleep(minPollingInterval)
		}
	}()

	return <-errChan
}

func (k *KingOfTheHillClient) Subscribe(id string) (<-chan *pumpfun.CoinData, error) {
	ch := make(chan *pumpfun.CoinData, bufferSize)

	k.listenersMu.Lock()
	defer k.listenersMu.Unlock()
	k.listeners[id] = ch

	return ch, nil
}

func (k *KingOfTheHillClient) Unsubscribe(id string) {
	k.listenersMu.Lock()
	defer k.listenersMu.Unlock()
	delete(k.listeners, id)
}

func (k *KingOfTheHillClient) notifyListeners(coinData *pumpfun.CoinData) {
	k.listenersMu.Lock()
	defer k.listenersMu.Unlock()

	for _, ch := range k.listeners {
		ch <- coinData
	}
}
