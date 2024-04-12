package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// CommandCenter struct
type CommandCenter struct {
	EtcdClient *clientv3.Client
	ID         string
	State      State `json:"state"`
}

type State struct {
	ID    string `json:"id"`
	Funds struct {
		Amount float32 `json:"amount"`
	} `json:"funds"`
	Firewall struct {
		Level float32 `json:"level"`
	} `json:"firewall"`
	Scanner struct {
		Level float32 `json:"level"`
	} `json:"scanner"`
	CryptoMiner struct {
		Level float32 `json:"level"`
	} `json:"cryptoMiner"`
	Stealer struct {
		Level float32 `json:"level"`
	} `json:"stealer"`
}

// UpgradeReply struct
type UpgradeReply struct {
	Allow bool    `json:"success"`
	Cost  float32 `json:"cost"`
}

// Handle setting of variables of env var is not set
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

// Handle setting of variables of env var is not set
func getEnvToArray(key, defaultValue string) []string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return strings.Split(defaultValue, ";")
	}
	return strings.Split(value, ";")
}

func (c *CommandCenter) Init(config clientv3.Config) CommandCenter {
	// Set player id from env variable
	c.ID = getEnv("ID", "123456")
	c.State.ID = c.ID
	c.State.CryptoMiner.Level = 1
	c.State.Scanner.Level = 1
	c.State.Firewall.Level = 1
	c.State.Stealer.Level = 1

	// Init etcd client
	cli, err := clientv3.New(config)
	// Throw error if not possible to init client
	if err != nil {
		log.Panicln("Could not setup etcd3 client.")
	}
	// Set the client in the struct
	c.EtcdClient = cli

	// Check if there is a state saved for the player id
	val := c.GetValue(c.ID)
	// Player not initialized. Initialize now
	if val == nil {
		// Convert struct to json via marshal
		state, err := json.Marshal(c.State)
		if err != nil {
			log.Panicln(err)
		}
		// Put value on etcd
		c.PutValue(c.ID, string(state))
		// Player already initialized. Get value and set it on command center
	} else {
		// Unmarshal value to struct
		err := json.Unmarshal(val, &c.State)
		if err != nil {
			log.Panicln("Could not unmarshal.")
		}
	}
	// Check for existing state now
	return *c
}

// Get value from etcd3
func (c *CommandCenter) GetValue(key string) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	resp, err := c.EtcdClient.Get(ctx, key)
	cancel()
	if err != nil {
		log.Panicln("Could not get key")
	}
	if resp.Count > 0 {
		return resp.Kvs[0].Value
	} else {
		return nil
	}
}

// Put valie to etcd3
func (c *CommandCenter) PutValue(key string, value string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, key, value)
	cancel()
	//fmt.Printf("Put value %s on topic: %s\n", value, key)
	if err != nil {
		log.Panicln("Could not put key")
	}
}

// Save state to etcd3 server to save state into
func (c *CommandCenter) SaveState() {
	value, marshalErr := json.Marshal(c.State)
	if marshalErr != nil {
		log.Panicf("Could not save state.\n")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, c.ID, string(value))
	cancel()
	//fmt.Printf("Put value %s on topic: %s\n", value, c.ID)
	if err != nil {
		log.Panicln("Could not put key")
	}
}

// Mine "crypto"
func (c *CommandCenter) Mine(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		minerLevel := c.State.CryptoMiner.Level
		c.State.Funds.Amount += baseMineRate * float32(minerLevel)
		c.SaveState()
		time.Sleep(time.Second * 1)
	}
}

// UpgradeStealer on CommandCenter
func (c *CommandCenter) UpgradeStealer(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeStealer", c.ID), []byte(state), time.Second)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	log.Printf("UpdateStealer reply: %s", msg.Data)
	jsonErr := json.Unmarshal(msg.Data, &reply)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
		return false, c, reply, err
	}
	if reply.Allow {
		c.State.Funds.Amount = c.State.Funds.Amount - reply.Cost
		c.State.Stealer.Level++
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeFirewall on CommandCenter
func (c *CommandCenter) UpgradeFirewall(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeFirewall", c.ID), []byte(state), time.Second)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	log.Printf("UpdateFirewall reply: %s", msg.Data)
	jsonErr := json.Unmarshal(msg.Data, &reply)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
		return false, c, reply, err
	}
	if reply.Allow {
		c.State.Funds.Amount = c.State.Funds.Amount - reply.Cost
		c.State.Firewall.Level++
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeScanner on CommandCenter
func (c *CommandCenter) UpgradeScanner(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeScanner", c.ID), []byte(state), time.Second)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	log.Printf("UpdateScanner reply: %s", msg.Data)
	jsonErr := json.Unmarshal(msg.Data, &reply)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
		return false, c, reply, err
	}
	if reply.Allow {
		c.State.Funds.Amount = c.State.Funds.Amount - reply.Cost
		c.State.Scanner.Level++
		return true, c, reply, err
	}
	return false, c, reply, nil
}

// UpgradeCryptoMiner on CommandCenter
func (c *CommandCenter) UpgradeCryptoMiner(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeMiner", c.ID), []byte(state), time.Second)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	log.Printf("UpdateMiner reply: %s", msg.Data)
	jsonErr := json.Unmarshal(msg.Data, &reply)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
		return false, c, reply, err
	}
	if reply.Allow {
		c.State.Funds.Amount = c.State.Funds.Amount - reply.Cost
		c.State.CryptoMiner.Level++
		return true, c, reply, nil
	}
	return false, c, reply, nil
}

// Subscribe to scan topic to reply to other players when they scan.
// This not reply when the incoming command center id is the same as the current instance
// When the foreign command center scanner level has a higher level than this command center's firewall, reply
func (c *CommandCenter) ReplyScan(nc *nats.Conn) error {
	if _, err := nc.Subscribe("scan", func(m *nats.Msg) {
		// Initialize CommandCenter struct
		foreignCommandCenter := State{}
		// Load received values
		err := json.Unmarshal(m.Data, &foreignCommandCenter)
		if err != nil {
			log.Fatalln(err)
		}
		// If the scanning command center is the same do not allow scan
		if c.ID == foreignCommandCenter.ID {
			return
		} else {

			// If the command center is foreign and their scanner level is higher than this firewall level, allow scan
			if foreignCommandCenter.Scanner.Level > c.State.Firewall.Level {
				jsonReply, err := json.Marshal(c.State)
				if err != nil {
					log.Fatalln(err)
				}
				m.Respond(jsonReply)
			}
			return
		}
	}); err != nil {
		return err
	}
	return nil
}

func (c *CommandCenter) RequestScan(nc *nats.Conn) ([]State, string, error) {
	// This costs money, so we remove coins based on the level of the scanner. First check if enough funds are available for scan.
	cost := c.State.Scanner.Level * 0.1
	if cost > c.State.Funds.Amount {
		err := fmt.Errorf("not enough funds. cost: %f", cost)
		return nil, fmt.Sprintf("Not enough funds. Cost: %f", cost), err
	} else {
		c.State.Funds.Amount = c.State.Funds.Amount - cost
	}

	var states = []State{}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return nil, "could not marshal json", err
	}
	log.Println("Subscribing to scan.")
	sub, err := nc.SubscribeSync(fmt.Sprintf("commandcenter.%s.scanreply", c.ID))
	if err != nil {
		log.Fatal(err)
	}
	nc.Flush()

	// Send the request
	log.Println("Publishing to scan.")
	// Publish scan event to game master
	nc.Publish("scanevent", []byte(state))
	nc.PublishRequest("scan", fmt.Sprintf("commandcenter.%s.scanreply", c.ID), []byte(state))

	// Wait for a moment and gather messages
	max := 100 * time.Millisecond
	start := time.Now()
	for time.Since(start) < max {
		log.Println("Iterate through messages")
		msg, err := sub.NextMsg(1 * time.Second)
		if err != nil {
			log.Println(err)
			break
		}
		st := &State{}
		json.Unmarshal(msg.Data, st)
		states = append(states, *st)
	}
	log.Println("Unsubscribing")
	sub.Unsubscribe()
	return states, "successful scan", nil
}

// Get state of CommandCenter
func (c *CommandCenter) GetState() *CommandCenter {
	return c
}
