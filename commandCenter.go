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
	StealData  struct {
		LastAttackTime time.Time
		AttackInterval time.Duration
	}
	State State `json:"state"`
}

type State struct {
	ID    string `json:"id"`
	Funds struct {
		Amount float32 `json:"amount"`
		*sync.RWMutex
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
	CoolDown struct {
		Time time.Duration `json:"time"`
	} `json:"cooldown"`
}

// UpgradeReply struct
type UpgradeReply struct {
	Allow bool    `json:"success"`
	Cost  float32 `json:"cost"`
}

// StealReply struct
type StealReply struct {
	Attacker    string  `json:"attacker"`
	Defender    string  `json:"defender"`
	Success     bool    `json:"success"`
	GainedCoins float32 `json:"gainedCoins"`
	CoolDown    bool    `json:"cooldown"`
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
	// Configure commandCenter State
	c.ID = getEnv("ID", "123456")
	c.State.ID = c.ID
	c.State.CryptoMiner.Level = 1
	c.State.Scanner.Level = 1
	c.State.Firewall.Level = 1
	c.State.Stealer.Level = 1
	// Set initial protection period on boot
	c.StealData.LastAttackTime = time.Now()
	// Configure starting attack interval
	c.StealData.AttackInterval = time.Minute * 5
	c.State.Funds.RWMutex = &sync.RWMutex{}

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

// Put value to etcd3
func (c *CommandCenter) PutValue(key string, value string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, key, value)
	cancel()
	//fmt.Printf("Put value %s on topic: %s\n", value, key)
	if err != nil {
		log.Panicln("Could not put key")
	}
}

// Delete value from etcd3
func (c *CommandCenter) DeleteState(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Delete(ctx, key)
	cancel()
	if err != nil {
		log.Panicln("Could not delete key")
		return err
	}
	return nil
}

// Deletes the state of the commandcenter from etcd3 and restarts the command center
func (c *CommandCenter) ListenReset(nc *nats.Conn) {
	if _, err := nc.Subscribe("reset", func(m *nats.Msg) {
		err := c.DeleteState(c.ID)
		if err != nil {
			log.Println("Could not reset command center")
			return
		}
		os.Exit(0)
	}); err != nil {
		log.Println(err)
	}
}

// Save state to etcd3 server
func (c *CommandCenter) SaveStateContinuous() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		value, marshalErr := json.Marshal(c.State)
		if marshalErr != nil {
			log.Panicf("Could not save state.\n")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := c.EtcdClient.Put(ctx, c.ID, string(value))
		cancel()
		if err != nil {
			log.Panicln("Could not put key")
		}
	}
}

// Save state to etcd3 server right now
func (c *CommandCenter) SaveStateImmediate() {
	value, marshalErr := json.Marshal(c.State)
	if marshalErr != nil {
		log.Panicf("Could not save state.\n")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, c.ID, string(value))
	cancel()
	if err != nil {
		log.Panicln("Could not put key")
	}

}

// Mine "crypto"
func (c *CommandCenter) Mine(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		minerLevel := c.State.CryptoMiner.Level
		c.State.Funds.Lock()
		c.State.Funds.Amount += baseMineRate * float32(minerLevel)
		c.State.Funds.Unlock()
		monitor.MinedCoins.WithLabelValues(c.ID).Add(float64(baseMineRate * float32(minerLevel)))
		time.Sleep(time.Second * 1)
	}
}

// UpgradeStealer on CommandCenter
func (c *CommandCenter) UpgradeStealer(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Funds.Lock()
	defer c.State.Funds.Unlock()
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
		monitor.SpentCoins.WithLabelValues(c.ID).Add(float64(reply.Cost))
		c.State.Stealer.Level++
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeFirewall on CommandCenter
func (c *CommandCenter) UpgradeFirewall(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Funds.Lock()
	defer c.State.Funds.Unlock()
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
		monitor.SpentCoins.WithLabelValues(c.ID).Add(float64(reply.Cost))
		c.State.Firewall.Level++
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeScanner on CommandCenter
func (c *CommandCenter) UpgradeScanner(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Funds.Lock()
	defer c.State.Funds.Unlock()
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
		monitor.SpentCoins.WithLabelValues(c.ID).Add(float64(reply.Cost))
		c.State.Scanner.Level++
		return true, c, reply, err
	}
	return false, c, reply, nil
}

// UpgradeCryptoMiner on CommandCenter
func (c *CommandCenter) UpgradeCryptoMiner(nc *nats.Conn) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Funds.Lock()
	defer c.State.Funds.Unlock()
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
		monitor.SpentCoins.WithLabelValues(c.ID).Add(float64(reply.Cost))
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

// Requests a scan from the network. This is a scatter-gather method.
// We subscribe to the reply topic synchronously, then publish a request to all commandcenters.
// Commandcenters all subscribe to scan by default. If the scanning command center has a scanner level higher than the firewall of others, they must respond.
func (c *CommandCenter) RequestScan(nc *nats.Conn) ([]State, string, error) {
	// This costs money, so we remove coins based on the level of the scanner. First check if enough funds are available for scan.
	cost := c.State.Scanner.Level * 0.1
	c.State.Funds.Lock()
	defer c.State.Funds.Unlock()
	if cost > c.State.Funds.Amount {
		err := fmt.Errorf("not enough funds. cost: %f", cost)
		return nil, fmt.Sprintf("Not enough funds. Cost: %f", cost), err
	} else {
		c.State.Funds.Amount = c.State.Funds.Amount - cost
		monitor.SpentCoins.WithLabelValues(c.ID).Add(float64(cost))
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
	// Publish scan event to game master
	nc.Publish("scanevent", []byte(state))
	nc.PublishRequest("scan", fmt.Sprintf("commandcenter.%s.scanreply", c.ID), []byte(state))

	// Wait for a moment and gather messages
	max := 100 * time.Millisecond
	start := time.Now()
	for time.Since(start) < max {
		msg, err := sub.NextMsg(1 * time.Second)
		if err != nil {
			log.Println(err)
			break
		}
		st := &State{}
		json.Unmarshal(msg.Data, st)
		states = append(states, *st)
	}
	sub.Unsubscribe()
	return states, "successful scan", nil
}

// Subscribe to steal topic to reply to other players when they attempt to steal.
// This will not reply when the incoming command center id is the same as the current instance
// When the foreign command center stealer level has a higher level than this command center's firewall, reply
func (c *CommandCenter) ReplySteal(nc *nats.Conn) error {
	if _, err := nc.Subscribe(fmt.Sprintf("commandcenter.%s.stealEndpoint", c.ID), func(m *nats.Msg) {
		log.Println("Incoming steal event.")
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
			// Check if there is a cooldown.
			if time.Since(c.StealData.LastAttackTime) < c.StealData.AttackInterval {
				stealReply := StealReply{
					Defender:    c.ID,
					Attacker:    foreignCommandCenter.ID,
					Success:     false,
					GainedCoins: 0,
					CoolDown:    true,
				}
				jsonReply, err := json.Marshal(stealReply)
				if err != nil {
					log.Fatalln(err)
				}
				m.Respond(jsonReply)
				return
			}
			// Set attack time because the defending command center is out of cooldown
			c.StealData.LastAttackTime = time.Now()
			// Stealer Level
			stealerLevel := foreignCommandCenter.Stealer.Level
			// Get current level difference
			lvldiff := int(stealerLevel) - int(c.State.Firewall.Level)
			log.Printf("Level diff: %d", lvldiff)
			if lvldiff <= 0 {
				// Reset the attack interval to the default 5 minutes
				c.StealData.AttackInterval = time.Minute * 5
				// Level of the stealer is not high enough. No coins stolen.
				stealReply := StealReply{
					Defender:    c.ID,
					Attacker:    foreignCommandCenter.ID,
					Success:     false,
					GainedCoins: 0,
					CoolDown:    false,
				}
				jsonReply, err := json.Marshal(stealReply)
				if err != nil {
					log.Fatalln(err)
				}
				m.Respond(jsonReply)
				return

			} else {
				// To balance strong players attacking weaker player:
				// The greater the level difference between attacker and defender the greater the cooldown the defender will receive, protecting them for longer.
				// Default will always be 5 minutes + level difference
				// Attacking weaker players will protect them for a long time, allowing them to catch up again.
				attackCooldown := 5 + lvldiff
				c.StealData.AttackInterval = time.Minute * time.Duration(attackCooldown)
				// Level of stealer is higher

				// Current Funds = 5.11
				// futureFunds   = 5
				// futureFunds   < 5.11? remove 0.11 from Funds and add to stealAttempts, update current Funds
				// Do not remove targetfunds if firewall level changes mid op.
				// Do not remove targetfunds if that would mean sinking below 0

				// Create a fixed size array based on the level difference
				stealAttempts := make([]float32, lvldiff)
				// Create a coin cache
				var coincache float32
				// Dyncamically determine amount to steal per attempt by level.
				stealAmount := float32(0.01) * stealerLevel
				log.Printf("Coins are being stolen! %f per attempt!", stealAmount)
				// Try to steal money on each iteration. Logic is described on top.
				c.State.Funds.Lock()
				for i := range stealAttempts {
					stealAttempts[i] = float32(i)
					//  futureFunds := c.State.Funds.Amount - stealAmount; futureFunds > c.State.Funds.Amount
					if c.State.Funds.Amount-stealAmount <= 0 {
						// Player loses no money because firewall level changed mid loop.
						if c.State.Firewall.Level >= stealerLevel {
							coincache += 0
							// Player all money and stealer gets the rest because the steal amount would cause a negative number.
						} else {
							coincache += c.State.Funds.Amount
							c.State.Funds.Amount = 0
							break
						}
					} else {
						// Steal money from command center
						c.State.Funds.Amount -= stealAmount
						// Store in cache
						coincache += stealAmount
					}
				}
				c.State.Funds.Unlock()
				monitor.LostCoins.WithLabelValues(c.ID, foreignCommandCenter.ID).Add(float64(coincache))
				// After successfully stealing from the target, send a reply to the attacker
				stealReply := StealReply{
					Defender:    c.ID,
					Attacker:    foreignCommandCenter.ID,
					Success:     true,
					GainedCoins: coincache,
					CoolDown:    false,
				}
				jsonReply, err := json.Marshal(stealReply)
				if err != nil {
					log.Fatalln(err)
				}
				m.Respond(jsonReply)
				return
			}
		}
	}); err != nil {
		return err
	}
	return nil
}

// Here we want to start a steal event. Compared to the scan, this only targets specific command centers by ID.
func (c *CommandCenter) StealFromTarget(targetId string, nc *nats.Conn) (StealReply, string, error) {
	var stealReply StealReply
	c.State.Funds.Lock()
	defer c.State.Funds.Unlock()
	// This costs money, so we remove coins based on the level of the stealer. First check if enough funds are available for scan.
	cost := c.State.Stealer.Level * 0.1
	if cost > c.State.Funds.Amount {
		err := fmt.Errorf("not enough funds. cost: %f", cost)
		return stealReply, fmt.Sprintf("Not enough funds. Cost: %f", cost), err
	} else {
		c.State.Funds.Amount = c.State.Funds.Amount - cost
		monitor.SpentCoins.WithLabelValues(c.ID).Add(float64(cost))
	}
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return stealReply, "could not marshal json", err
	}
	// Subscribe temporarily to a created stealReply topic to listen for responses from the target
	sub, err := nc.SubscribeSync(fmt.Sprintf("commandcenter.%s.stealReply", c.ID))
	if err != nil {
		log.Fatal(err)
	}
	nc.Flush()
	// Publish steal event to game master
	nc.Publish("stealevent", []byte(state))
	nc.PublishRequest(fmt.Sprintf("commandcenter.%s.stealEndpoint", targetId), fmt.Sprintf("commandcenter.%s.stealReply", c.ID), []byte(state))

	// Wait for a single response
	for {
		msg, err := sub.NextMsg(5 * time.Second)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal(msg.Data, &stealReply)
		// Send stealreply data to stealresult topic to let others know who stole from whom
		nc.Publish("stealresult", msg.Data)
		nc.Flush()
		break
	}
	sub.Unsubscribe()
	// Add coins to account. If no coins are gained, then nothing will be added.
	c.State.Funds.Amount += stealReply.GainedCoins
	monitor.StolenCoins.WithLabelValues(c.ID).Add(float64(stealReply.GainedCoins))
	// Return reply with error on cooldown
	if stealReply.CoolDown {
		err := fmt.Errorf("target is on cooldown")
		return stealReply, "Target is on cooldown", err
	}

	return stealReply, "successful steal operation", nil
}

// Periodically update the ticker
func (c *CommandCenter) UpdateCoolDown() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		if c.StealData.AttackInterval/time.Second-(time.Since(c.StealData.LastAttackTime)/time.Second) > 0 {
			c.State.CoolDown.Time = c.StealData.AttackInterval/time.Second - (time.Since(c.StealData.LastAttackTime) / time.Second)
			monitor.CoolDown.WithLabelValues(c.ID).Set(float64(c.State.CoolDown.Time))
		} else {
			c.State.CoolDown.Time = 0
			monitor.CoolDown.WithLabelValues(c.ID).Set(float64(c.State.CoolDown.Time))
		}
	}
}

// Get state of CommandCenter
func (c *CommandCenter) GetState() *CommandCenter {
	return c
}
