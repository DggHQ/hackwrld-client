package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// CommandCenter struct
type CommandCenter struct {
	EtcdClient *clientv3.Client
	ID         string
	State      struct {
		ID    string `json:"id"`
		Funds struct {
			Amount float32 `json:"amount"`
		} `json:"funds"`
		Firewall struct {
			Level int `json:"level"`
		} `json:"firewall"`
		Scanner struct {
			Level int `json:"level"`
		} `json:"scanner"`
		CryptoMiner struct {
			Level int `json:"level"`
		} `json:"cryptoMiner"`
	} `json:"state"`
}

// Firewall   struct {
// 	Level int `json:"level"`
// } `json:"firewall"`
// Funds struct {
// 	Amount float32 `json:"amount"`
// } `json:"funds"`
// Scanner struct {
// 	Level int `json:"level"`
// } `json:"scanner"`
// CryptoMiner struct {
// 	Level int `json:"level"`
// } `json:"cryptoMiner"`

// UpgradeReply struct
type UpgradeReply struct {
	Allow bool
	Cost  float32
}

func (c *CommandCenter) Init(config clientv3.Config) CommandCenter {
	// Set player id from env variable TODO: Env variable
	c.ID = "123456"
	c.State.ID = c.ID
	c.State.CryptoMiner.Level = 1

	println(c.State.CryptoMiner.Level)

	// Init etcd client
	cli, err := clientv3.New(config)
	// Throw error if not possible to init client
	if err != nil {
		log.Panicln("Could not setup etcd3 client.")
	}
	// Close connection when everything is done
	//defer cli.Close()
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

func (c *CommandCenter) PutValue(key string, value string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, key, value)
	cancel()
	fmt.Printf("Put value %s on topic: %s\n", value, key)
	if err != nil {
		log.Panicln("Could not put key")
	}
}

func (c *CommandCenter) SaveState() {
	value, marshalErr := json.Marshal(c.State)
	if marshalErr != nil {
		log.Panicf("Could not save state.\n")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, c.ID, string(value))
	cancel()
	fmt.Printf("Put value %s on topic: %s\n", value, c.ID)
	if err != nil {
		log.Panicln("Could not put key")
	}
}

// // Prepare Command Center
// func (c *CommandCenter) Prepare(savedCommandCenter CommandCenter) CommandCenter {

// 	// Configure ID
// 	c.ID = savedCommandCenter.ID
// 	if c.ID == "" {
// 		uuidWithHyphen := uuid.New()
// 		uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)
// 		c.ID = uuid
// 	}
// 	// Configure Funds
// 	c.Funds = savedCommandCenter.Funds
// 	// Confirue Firewall
// 	c.Firewall = savedCommandCenter.Firewall
// 	// Configure Scanner
// 	c.Scanner = savedCommandCenter.Scanner
// 	// Configure CryptoMiner
// 	c.CryptoMiner = savedCommandCenter.CryptoMiner
// 	return *c
// }

// // Save state of Command Center
// func (c *CommandCenter) Save() error {
// 	jsonConfig, err := json.Marshal(c)
// 	if err != nil {
// 		return err
// 	}
// 	ioutil.WriteFile("config.json", jsonConfig, os.ModePerm)
// 	return nil
// }

// Mine crypto
func (c *CommandCenter) Mine(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		minerLevel := c.State.CryptoMiner.Level
		c.State.Funds.Amount += baseMineRate * float32(minerLevel)
		c.SaveState()
		// err := c.Save()
		// if err != nil {
		// 	log.Fatalln(err)
		// }
		time.Sleep(time.Second * 1)
	}
}

// // PublishState CommandCenter State
// func (c *CommandCenter) PublishState(nc *nats.Conn, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	for {
// 		state, err := json.Marshal(&c)
// 		if err != nil {
// 			log.Fatalln(err)
// 		}
// 		if err := nc.Publish(fmt.Sprintf("commandcenter.%s.state", c.ID), []byte(state)); err != nil {
// 			log.Fatal(err)
// 		}
// 		time.Sleep(time.Second * 10)
// 	}
// }

// // UpgradeFirewall on CommandCenter
// func (c *CommandCenter) UpgradeFirewall(nc *nats.Conn) (bool, error) {
// 	reply := UpgradeReply{}
// 	state, err := json.Marshal(&c)
// 	if err != nil {
// 		log.Fatalln(err)
// 		return false, err
// 	}
// 	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeFirewall", c.ID), []byte(state), time.Second)
// 	if err != nil {
// 		log.Fatalln(err)
// 		return false, err
// 	}
// 	log.Printf("UpdateFirewall reply: %s", msg.Data)
// 	jsonErr := json.Unmarshal(msg.Data, &reply)
// 	if jsonErr != nil {
// 		log.Fatalln(jsonErr)
// 		return false, err
// 	}
// 	if reply.Allow {
// 		c.Funds.Amount = c.Funds.Amount - reply.Cost
// 		c.Firewall.Level++
// 		return true, nil
// 	}
// 	return false, nil
// 	//c.Firewall.Level++
// }

// // UpgradeScanner on CommandCenter
// func (c *CommandCenter) UpgradeScanner(nc *nats.Conn) (bool, error) {
// 	reply := UpgradeReply{}
// 	state, err := json.Marshal(&c)
// 	if err != nil {
// 		log.Fatalln(err)
// 		return false, err
// 	}
// 	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeScanner", c.ID), []byte(state), time.Second)
// 	if err != nil {
// 		log.Fatalln(err)
// 		return false, err
// 	}
// 	log.Printf("UpdateScanner reply: %s", msg.Data)
// 	jsonErr := json.Unmarshal(msg.Data, &reply)
// 	if jsonErr != nil {
// 		log.Fatalln(jsonErr)
// 		return false, err
// 	}
// 	if reply.Allow {
// 		c.Funds.Amount = c.Funds.Amount - reply.Cost
// 		c.Scanner.Level++
// 		return true, nil
// 	}
// 	return false, nil
// 	//c.Scanner.Level++
// }

// UpgradeCryptoMiner on CommandCenter
func (c *CommandCenter) UpgradeCryptoMiner(nc *nats.Conn) (bool, error) {
	reply := UpgradeReply{}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, err
	}
	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeMiner", c.ID), []byte(state), time.Second)
	if err != nil {
		log.Fatalln(err)
		return false, err
	}
	log.Printf("UpdateMiner reply: %s", msg.Data)
	jsonErr := json.Unmarshal(msg.Data, &reply)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
		return false, err
	}
	if reply.Allow {
		c.State.Funds.Amount = c.State.Funds.Amount - reply.Cost
		c.State.CryptoMiner.Level++
		return true, nil
	}
	return false, nil

	//c.CryptoMiner.Level++
}
