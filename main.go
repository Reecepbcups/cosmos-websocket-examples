package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "cosmoshub.rpc.kjnodes.com:443", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// ! If the node is not :443 (i.e. localhost or direct IP), use ws instead of wss
	u := url.URL{Scheme: "wss", Host: *addr, Path: "/websocket"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			// Read in the JSON response from the server
			var out json.RawMessage
			if err := c.ReadJSON(&out); err != nil {
				log.Println("read:", err)
				return
			}

			// decode the bytes into the block type
			var b Block
			if err := json.Unmarshal(out, &b); err != nil {
				log.Println("unmarshal:", err)
				return
			}

			height := b.Result.Data.Value.Block.Header.Height
			if height == "" {
				fmt.Println("new connection made to node")
				continue
			}
			log.Println("block height:", height)

			/*
				NOTE: if you want to decode this tx into a human readable format, you can use the app binary:
				use the RPC endpoint to decode

				curl -H "Content-type: application/json" -d '{
				"tx_bytes":"ClUKUwobL2Nvc21vcy5nb3YudjFiZXRhMS5Nc2dWb3RlEjQI+gYSLWNvc21vczFtYWRjemhwZjAzdGg4MHc3a25jZTg2YWRzanRobHJhZjNuMjkzeBgBEmcKUApGCh8vY29zbW9zLmNyeXB0by5zZWNwMjU2azEuUHViS2V5EiMKIQLR5x0Usgnw1PI3oFeKOaxcwYI/kbRVJZ8UXtM63CsF2RIECgIIfxhxEhMKDQoFdWF0b20SBDE0NTEQr8UDGkCHeTNhE8zLTK3To4hFvV4iP7ZwtXUSwDLsFrGwRhhWXSHujXEneMixrfZ+v6VxAoV/X3GQ1i0MdEW+Lb40eLgK"
				}' 'https://cosmoshub.api.kjnodes.com/cosmos/tx/v1beta1/decode'


				or:
				gaiad tx decode ClUKUwobL2Nvc21vcy5nb3YudjFiZXRhMS5Nc2dWb3RlEjQIkAcSLWNvc21vczF6N2hzbDVxbmdhMmMwNnp2ZTVsNTVoZjMzcXlqMmNxMGFtOHZ2aBgBEmcKUApGCh8vY29zbW9zLmNyeXB0by5zZWNwMjU2azEuUHViS2V5EiMKIQM/S+TZUfnNw2l5YdZOHV9Bt37RD8vVQp9D4iuzJf4+yBIECgIIfxhEEhMKDQoFdWF0b20SBDE2NjQQ84cEGkDEGl3M7x+sNNl9sWllpYKw/4WaaQGTTDy5Bt3FYRgjtnMciHSJUil5qxEXwkSiZNe3k/98zkfxWEMQz5fMOWS3

				or: import the app github and decode directly using the Cosmos apps interface registry types
			*/
			txs := make(map[string]string)
			for _, tx := range b.Result.Data.Value.Block.Data.Txs {
				tx := tx
				// decodes the RPC's base64 format -> the raw proto bytes
				decodedData, err := base64.StdEncoding.DecodeString(tx)
				if err != nil {
					log.Println("decode:", err)
					return
				}

				// sha256 hash the raw bytes into the tx hash
				txHash := fmt.Sprintf("%x", sha256.Sum256(decodedData))
				// fmt.Println("txHash:", strings.ToUpper(txHash))

				txs[txHash] = tx
			}

			for txHash := range txs {
				fmt.Println("txs hash:", txHash)
			}

		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// subscribe to the cosmos node
	// '{"jsonrpc": "2.0", "method": "subscribe", "params": ["tm.event=\'NewBlock\'"], "id": 1}'
	err = c.WriteJSON(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "subscribe",
		"params":  []string{"tm.event='NewBlock'"},
		"id":      1,
	})
	if err != nil {
		log.Println("write:", err)
		return
	}

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

// Copy pasted a subscribed block output JSON format into https://mholt.github.io/json-to-go/
type Block struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Query string `json:"query"`
		Data  struct {
			Type  string `json:"type"`
			Value struct {
				Block struct {
					Header struct {
						Version struct {
							Block string `json:"block"`
						} `json:"version"`
						ChainID     string    `json:"chain_id"`
						Height      string    `json:"height"`
						Time        time.Time `json:"time"`
						LastBlockID struct {
							Hash  string `json:"hash"`
							Parts struct {
								Total int    `json:"total"`
								Hash  string `json:"hash"`
							} `json:"parts"`
						} `json:"last_block_id"`
						LastCommitHash     string `json:"last_commit_hash"`
						DataHash           string `json:"data_hash"`
						ValidatorsHash     string `json:"validators_hash"`
						NextValidatorsHash string `json:"next_validators_hash"`
						ConsensusHash      string `json:"consensus_hash"`
						AppHash            string `json:"app_hash"`
						LastResultsHash    string `json:"last_results_hash"`
						EvidenceHash       string `json:"evidence_hash"`
						ProposerAddress    string `json:"proposer_address"`
					} `json:"header"`
					Data struct {
						Txs []string `json:"txs"`
					} `json:"data"`
					Evidence struct {
						Evidence []any `json:"evidence"`
					} `json:"evidence"`
					LastCommit struct {
						Height  string `json:"height"`
						Round   int    `json:"round"`
						BlockID struct {
							Hash  string `json:"hash"`
							Parts struct {
								Total int    `json:"total"`
								Hash  string `json:"hash"`
							} `json:"parts"`
						} `json:"block_id"`
						Signatures []struct {
							BlockIDFlag      int       `json:"block_id_flag"`
							ValidatorAddress string    `json:"validator_address"`
							Timestamp        time.Time `json:"timestamp"`
							Signature        string    `json:"signature"`
						} `json:"signatures"`
					} `json:"last_commit"`
				} `json:"block"`
				ResultBeginBlock struct {
					Events []struct {
						Type       string `json:"type"`
						Attributes []struct {
							Key   string `json:"key"`
							Value string `json:"value"`
							Index bool   `json:"index"`
						} `json:"attributes"`
					} `json:"events"`
				} `json:"result_begin_block"`
				ResultEndBlock struct {
					ValidatorUpdates []struct {
						PubKey struct {
							Sum struct {
								Type  string `json:"type"`
								Value struct {
									Ed25519 string `json:"ed25519"`
								} `json:"value"`
							} `json:"Sum"`
						} `json:"pub_key"`
						Power string `json:"power"`
					} `json:"validator_updates"`
					ConsensusParamUpdates struct {
						Block struct {
							MaxBytes string `json:"max_bytes"`
							MaxGas   string `json:"max_gas"`
						} `json:"block"`
						Evidence struct {
							MaxAgeNumBlocks string `json:"max_age_num_blocks"`
							MaxAgeDuration  string `json:"max_age_duration"`
							MaxBytes        string `json:"max_bytes"`
						} `json:"evidence"`
						Validator struct {
							PubKeyTypes []string `json:"pub_key_types"`
						} `json:"validator"`
					} `json:"consensus_param_updates"`
					Events []struct {
						Type       string `json:"type"`
						Attributes []struct {
							Key   string `json:"key"`
							Value string `json:"value"`
							Index bool   `json:"index"`
						} `json:"attributes"`
					} `json:"events"`
				} `json:"result_end_block"`
			} `json:"value"`
		} `json:"data"`
		Events struct {
			SendPacketPacketConnection       []string `json:"send_packet.packet_connection"`
			TmEvent                          []string `json:"tm.event"`
			CoinReceivedAmount               []string `json:"coin_received.amount"`
			MessageSender                    []string `json:"message.sender"`
			MintBondedRatio                  []string `json:"mint.bonded_ratio"`
			SendPacketPacketDstPort          []string `json:"send_packet.packet_dst_port"`
			CoinSpentAmount                  []string `json:"coin_spent.amount"`
			MintInflation                    []string `json:"mint.inflation"`
			SendPacketPacketData             []string `json:"send_packet.packet_data"`
			RewardsValidator                 []string `json:"rewards.validator"`
			SendPacketPacketTimeoutHeight    []string `json:"send_packet.packet_timeout_height"`
			SendPacketPacketChannelOrdering  []string `json:"send_packet.packet_channel_ordering"`
			SendPacketConnectionID           []string `json:"send_packet.connection_id"`
			SendPacketPacketTimeoutTimestamp []string `json:"send_packet.packet_timeout_timestamp"`
			SendPacketPacketDstChannel       []string `json:"send_packet.packet_dst_channel"`
			TransferRecipient                []string `json:"transfer.recipient"`
			TransferAmount                   []string `json:"transfer.amount"`
			MintAnnualProvisions             []string `json:"mint.annual_provisions"`
			CommissionAmount                 []string `json:"commission.amount"`
			CoinbaseAmount                   []string `json:"coinbase.amount"`
			CoinSpentSpender                 []string `json:"coin_spent.spender"`
			MessageModule                    []string `json:"message.module"`
			MintAmount                       []string `json:"mint.amount"`
			CommissionValidator              []string `json:"commission.validator"`
			RewardsAmount                    []string `json:"rewards.amount"`
			SendPacketPacketDataHex          []string `json:"send_packet.packet_data_hex"`
			SendPacketPacketSrcPort          []string `json:"send_packet.packet_src_port"`
			SendPacketPacketSrcChannel       []string `json:"send_packet.packet_src_channel"`
			CoinReceivedReceiver             []string `json:"coin_received.receiver"`
			CoinbaseMinter                   []string `json:"coinbase.minter"`
			TransferSender                   []string `json:"transfer.sender"`
			SendPacketPacketSequence         []string `json:"send_packet.packet_sequence"`
		} `json:"events"`
	} `json:"result"`
}
