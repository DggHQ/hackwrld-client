package main

// gpt71 id: 180887
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
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
	Nick       string
	Team       string
	StealData  struct {
		LastAttackTime time.Time
		AttackInterval time.Duration
	}
	ScrambledTimeStamp int64
	State              State `json:"state"`
}

type State struct {
	ID    string `json:"id"`
	Nick  string `json:"nick"`
	Team  string `json:"team"`
	Funds struct {
		Amount float32 `json:"amount"`
		*sync.RWMutex
	} `json:"funds"`
	Vault struct {
		Amount   float32 `json:"amount"`
		Level    float32 `json:"level"`
		Capacity float32 `json:"capacity"`
		*sync.RWMutex
	} `json:"vault"`
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
		// Time                time.Duration `json:"time"`
		ExpirationTimeStamp int64 `json:"expirationTimestamp"`
	} `json:"cooldown"`
	LastSteals []StealData `json:"lastSteals"`
	Inventory  struct {
		VaultMiner struct {
			AmountLeft float32 `json:"amountLeft"`
			Enabled    bool    `json:"enabled"`
		} `json:"vaultMiner"`
		ScanScrambler struct {
			AmountLeft float32 `json:"amountLeft"`
			Enabled    bool    `json:"enabled"`
		} `json:"scanScrambler"`
		PanicTransfer struct {
			AmountLeft float32 `json:"amountLeft"`
			Enabled    bool    `json:"enabled"`
		} `json:"panicTransfer"`
		NFT struct {
			Amount int `json:"amount"`
		} `json:"nft"`
	} `json:"inventory"`
}

// type Inventory struct {
// 	VaultMiner struct {
// 		AmountLeft float32 `json:"amountLeft"`
// 		Enabled    bool    `json:"enabled"`
// 	} `json:"vaultMiner"`
// }

// Hold data pertaining to last stealer
type StealData struct {
	Nick   string  `json:"nick"`
	Amount float32 `json:"amount"`
}

// UpgradeReply struct
type UpgradeReply struct {
	Allow  bool    `json:"success"`
	Cost   float32 `json:"cost"`
	Levels float32 `json:"levels"`
}

// StealReply struct
type StealReply struct {
	Attacker struct {
		ID   string `json:"id"`
		Nick string `json:"nick"`
	} `json:"attacker"`
	Defender struct {
		ID   string `json:"id"`
		Nick string `json:"nick"`
	} `json:"defender"`
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

// Get the current player level
func (c *CommandCenter) GetPlayerLevel() float32 {
	return c.State.Stealer.Level + c.State.Firewall.Level + c.State.CryptoMiner.Level + c.State.Scanner.Level
}

// Enables the VaultMiner.
func (c *CommandCenter) ActivateVaultMiner() (bool, error) {
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	// Do not activate when already enabled
	if c.State.Inventory.VaultMiner.Enabled {
		return false, errors.New("VaultMiner is already activated.")
	}
	// Calculate cost and allow / deny based on funds available
	cost := c.State.Vault.Capacity * 0.2
	if cost > c.State.Vault.Amount {
		return false, fmt.Errorf("Could not activate VaultMiner. Costs %f", cost)
	}
	c.State.Vault.Amount -= cost
	monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(cost))
	c.State.Inventory.VaultMiner.AmountLeft = c.State.Vault.Capacity * 0.75
	c.State.Inventory.VaultMiner.Enabled = true
	return true, nil
}

func (c *CommandCenter) ActivatePanicTransfer() (bool, error) {
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	// Do not activate when already enabled
	if c.State.Inventory.PanicTransfer.Enabled {
		return false, errors.New("PanicTransfer is already activated.")
	}
	// Calculate cost and allow / deny based on funds available
	cost := c.GetPlayerLevel() / 10
	if cost > c.State.Vault.Amount {
		return false, fmt.Errorf("Could not activate PanicTransfer. Costs %f", cost)
	}
	c.State.Vault.Amount -= cost
	monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(cost))
	c.State.Inventory.PanicTransfer.AmountLeft = float32(math.Ceil(float64(c.GetPlayerLevel()) / 20))
	c.State.Inventory.PanicTransfer.Enabled = true
	return true, nil
}

func (c *CommandCenter) PanicTransfer() {
	// Check if this panictransfer would cause the amountleft to sink to 0 or greater than 0
	if (c.State.Inventory.PanicTransfer.AmountLeft - 1) >= 0 {
		// Remove 1 from amount left if vault is not full
		if c.State.Vault.Amount != c.State.Vault.Capacity {
			c.State.Inventory.PanicTransfer.AmountLeft -= 1
			// Store in Vault
			log.Println("PanicTransfer Triggered.")
			c.StoreVault()
		}
		// If amount left after transfer is 0 disable panictransfer
		if c.State.Inventory.PanicTransfer.AmountLeft == 0 {
			c.State.Inventory.PanicTransfer.Enabled = false
		}
	}
}

func (c *CommandCenter) ActivateScanScrambler() (bool, error) {
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	// Do not activate when already enabled
	if c.State.Inventory.ScanScrambler.Enabled {
		return false, errors.New("ScanScrambler is already activated.")
	}
	// Calculate cost and allow / deny based on funds available
	cost := c.GetPlayerLevel() / 5
	if cost > c.State.Vault.Amount {
		return false, fmt.Errorf("Could not activate ScanScrambler. Costs %f", cost)
	}
	c.State.Vault.Amount -= cost
	monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(cost))
	c.State.Inventory.ScanScrambler.AmountLeft = float32(math.Ceil(float64(c.GetPlayerLevel()) / 50))
	c.State.Inventory.ScanScrambler.Enabled = true
	return true, nil
}

func (c *CommandCenter) ScanScramble() {
	// Check if this scramble would cause the amountleft to sink to 0 or greater than 0
	if (c.State.Inventory.ScanScrambler.AmountLeft - 1) >= 0 {

		// Remove 1 from amount left
		c.State.Inventory.ScanScrambler.AmountLeft -= 1
		// Generate random unix timestamp in the future between 5 and 10 minutes
		max := 600
		min := 300
		randomTime := time.Duration(rand.Intn(max-min)+min) * time.Second
		scrambledTime := int64(time.Now().Add(randomTime).Unix())
		log.Printf("Sent random time: %d", scrambledTime)
		c.ScrambledTimeStamp = scrambledTime
		// If amount left after transfer is 0 disable scrambler
		if c.State.Inventory.ScanScrambler.AmountLeft == 0 {
			c.State.Inventory.ScanScrambler.Enabled = false
		}
	}
}

// Mine to vault
// If the mined coin is larger than the amount left on the vault miner then just send the difference to the vault.
// Set Enabled to false if AmountLeft is 0
func (c *CommandCenter) MineCoinToVault(minedCoin float32) {
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	if minedCoin > c.State.Inventory.VaultMiner.AmountLeft {
		c.State.Vault.Amount += minedCoin - c.State.Inventory.VaultMiner.AmountLeft
		c.State.Inventory.VaultMiner.AmountLeft = 0
		c.State.Inventory.VaultMiner.Enabled = false
	} else {
		c.State.Vault.Amount += minedCoin
		c.State.Inventory.VaultMiner.AmountLeft -= minedCoin
	}
}

// Update StealList of State
// This will show the last 50 steals in the state
func (c *CommandCenter) UpdateStealList(stealerName string, stolenAmount float32) {
	steal := StealData{
		Nick:   stealerName,
		Amount: stolenAmount,
	}
	// Remove oldest element of slice of there are greater or equal steals in state
	if len(c.State.LastSteals) >= 50 {
		c.State.LastSteals = c.State.LastSteals[1:]
	}
	// Append the steal to the state
	c.State.LastSteals = append(c.State.LastSteals, steal)
}

// Set Steal Cooldown
// Set to a fixed time of 10 Minutes after a steal
func (c *CommandCenter) SetCooldown() {
	// attackCooldown := 5 + levelDifference
	// if time.Minute*time.Duration(attackCooldown) > time.Hour*time.Duration(24) {
	// 	c.StealData.AttackInterval = time.Hour * time.Duration(24)
	// } else {
	// 	c.StealData.AttackInterval = time.Minute * time.Duration(attackCooldown)
	// }
	c.StealData.AttackInterval = time.Minute * 10
	c.State.CoolDown.ExpirationTimeStamp = int64(time.Now().Add(c.StealData.AttackInterval).Unix())
}

func (c *CommandCenter) Init(config clientv3.Config) CommandCenter {
	log.Printf("Starting command center for %s", getEnv("NICK", "DEBUGPLAYER"))
	// Configure commandCenter State
	c.ID = getEnv("ID", "123456")
	c.Nick = getEnv("NICK", "DEBUGPLAYER")
	c.Team = getEnv("TEAM", "none")
	c.State.ID = c.ID
	c.State.Nick = c.Nick
	c.State.Team = c.Team
	c.State.CryptoMiner.Level = 1
	c.State.Scanner.Level = 1
	c.State.Firewall.Level = 1
	c.State.Stealer.Level = 1
	c.State.Vault.Level = 1
	c.State.Vault.Capacity = 10
	c.ScrambledTimeStamp = 0
	// Set initial protection period on boot
	c.StealData.LastAttackTime = time.Now()
	// Configure starting attack interval
	c.StealData.AttackInterval = time.Minute * 5
	// Set initial gui cooldown time
	c.State.CoolDown.ExpirationTimeStamp = int64(time.Now().Add(c.StealData.AttackInterval).Unix())
	c.State.Funds.RWMutex = &sync.RWMutex{}
	c.State.Vault.RWMutex = &sync.RWMutex{}
	c.State.LastSteals = []StealData{}
	// Initialize Inventory
	c.State.Inventory.VaultMiner.AmountLeft = 0
	c.State.Inventory.VaultMiner.Enabled = false
	c.State.Inventory.PanicTransfer.AmountLeft = 0
	c.State.Inventory.PanicTransfer.Enabled = false
	c.State.Inventory.ScanScrambler.AmountLeft = 0
	c.State.Inventory.ScanScrambler.Enabled = false
	c.State.Inventory.NFT.Amount = 0

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

// Store coins in vault
func (c *CommandCenter) StoreVault() float32 {
	c.State.Vault.Lock()
	c.State.Funds.Lock()
	defer c.State.Vault.Unlock()
	defer c.State.Funds.Unlock()
	// Transfer money to vault but do not exceed the max capacity.
	transferAmount := c.State.Funds.Amount
	// Check if adding the current liquid funds would exceed the capacity of the vault.
	if c.State.Vault.Amount+transferAmount > c.State.Vault.Capacity {
		amountToAdd := c.State.Vault.Capacity - c.State.Vault.Amount
		c.State.Vault.Amount = c.State.Vault.Capacity
		c.State.Funds.Amount -= amountToAdd
		return amountToAdd
	}
	c.State.Vault.Amount += transferAmount
	// Empty the Funds
	c.State.Funds.Amount = 0
	return transferAmount
}

// Store coins in vault
func (c *CommandCenter) WithdrawVault() float32 {
	c.State.Vault.Lock()
	c.State.Funds.Lock()
	defer c.State.Vault.Unlock()
	defer c.State.Funds.Unlock()
	// Withdraw all coins from vault and set vault amount to 0
	transferAmount := c.State.Vault.Amount
	c.State.Funds.Amount += transferAmount
	c.State.Vault.Amount = 0
	return transferAmount
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
		log.Println("Could not put key")
	}
}

// Delete value from etcd3
func (c *CommandCenter) DeleteState(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Delete(ctx, key)
	cancel()
	if err != nil {
		log.Println("Could not delete key")
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
			log.Println("Could not save state.")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := c.EtcdClient.Put(ctx, c.ID, string(value))
		cancel()
		if err != nil {
			log.Println("Could not put key. Trying again in a minute")
		}
	}
}

// Save state to etcd3 server right now
func (c *CommandCenter) SaveStateImmediate() {
	value, marshalErr := json.Marshal(c.State)
	if marshalErr != nil {
		log.Println("Could not save state.")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := c.EtcdClient.Put(ctx, c.ID, string(value))
	cancel()
	if err != nil {
		log.Println("Could not put key.")
	}

}

// Mine "crypto"
func (c *CommandCenter) Mine(wg *sync.WaitGroup) {
	defer wg.Done()
	for {

		minerLevel := c.State.CryptoMiner.Level
		if c.State.Inventory.VaultMiner.Enabled && (c.State.Vault.Amount < c.State.Vault.Capacity) {
			// Check if VaultMiner is activated and if the vault is currently full
			minedCoin := baseMineRate * float32(minerLevel)
			if c.State.Vault.Amount+minedCoin > c.State.Vault.Capacity && (c.State.Vault.Amount != c.State.Vault.Capacity) {
				// diff := (c.State.Vault.Amount + minedCoin) - c.State.Vault.Capacity
				diff := c.State.Vault.Capacity - c.State.Vault.Amount
				c.MineCoinToVault(diff)
			} else {
				c.MineCoinToVault(minedCoin)
			}
		} else {
			// If not then just mine to "unsafe" wallet
			c.State.Funds.Lock()
			c.State.Funds.Amount += baseMineRate * float32(minerLevel)
			c.State.Funds.Unlock()
		}
		monitor.MinedCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(baseMineRate * float32(minerLevel)))
		time.Sleep(time.Second * 1)
	}
}

// UpgradeVault on CommandCenter this will not have a max buy option since the vault upgrade costs as much as its capacity
func (c *CommandCenter) UpgradeVault(nc *nats.Conn, upgradeToMax bool) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	msg, err := nc.Request(fmt.Sprintf("commandcenter.%s.upgradeVault", c.ID), []byte(state), time.Second)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	log.Printf("UpdateVault reply: %s", msg.Data)
	jsonErr := json.Unmarshal(msg.Data, &reply)
	if jsonErr != nil {
		log.Fatalln(jsonErr)
		return false, c, reply, err
	}
	if reply.Allow {
		c.State.Vault.Amount = c.State.Vault.Amount - reply.Cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(reply.Cost))
		c.State.Vault.Level += reply.Levels
		c.State.Vault.Capacity *= 2
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeStealer on CommandCenter
func (c *CommandCenter) UpgradeStealer(nc *nats.Conn, upgradeToMax bool) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	var msg *nats.Msg
	if upgradeToMax {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeStealer.max", c.ID), []byte(state), time.Second)
	} else {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeStealer", c.ID), []byte(state), time.Second)
	}
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
		c.State.Vault.Amount = c.State.Vault.Amount - reply.Cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(reply.Cost))
		c.State.Stealer.Level += reply.Levels
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeFirewall on CommandCenter
func (c *CommandCenter) UpgradeFirewall(nc *nats.Conn, upgradeToMax bool) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	var msg *nats.Msg
	if upgradeToMax {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeFirewall.max", c.ID), []byte(state), time.Second)
	} else {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeFirewall", c.ID), []byte(state), time.Second)
	}
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
		c.State.Vault.Amount = c.State.Vault.Amount - reply.Cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(reply.Cost))
		c.State.Firewall.Level += reply.Levels
		return true, c, reply, err
	}
	return false, c, reply, err
}

// UpgradeScanner on CommandCenter
func (c *CommandCenter) UpgradeScanner(nc *nats.Conn, upgradeToMax bool) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	var msg *nats.Msg
	if upgradeToMax {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeScanner.max", c.ID), []byte(state), time.Second)
	} else {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeScanner", c.ID), []byte(state), time.Second)
	}
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
		c.State.Vault.Amount = c.State.Vault.Amount - reply.Cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(reply.Cost))
		c.State.Scanner.Level += reply.Levels
		return true, c, reply, err
	}
	return false, c, reply, nil
}

// UpgradeCryptoMiner on CommandCenter
func (c *CommandCenter) UpgradeCryptoMiner(nc *nats.Conn, upgradeToMax bool) (bool, *CommandCenter, UpgradeReply, error) {
	reply := UpgradeReply{}
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	state, err := json.Marshal(&c.State)
	if err != nil {
		log.Fatalln(err)
		return false, c, reply, err
	}
	var msg *nats.Msg
	if upgradeToMax {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeMiner.max", c.ID), []byte(state), time.Second)
	} else {
		msg, err = nc.Request(fmt.Sprintf("commandcenter.%s.upgradeMiner", c.ID), []byte(state), time.Second)
	}
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
		c.State.Vault.Amount = c.State.Vault.Amount - reply.Cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(reply.Cost))
		c.State.CryptoMiner.Level += reply.Levels
		return true, c, reply, nil
	}
	return false, c, reply, nil
}

// Send playerinfo for leaderboard container
func (c *CommandCenter) SendPlayerInfo(nc *nats.Conn) error {
	if _, err := nc.Subscribe(fmt.Sprintf("commandcenter.%s.info", c.ID), func(m *nats.Msg) {
		jsonReply, err := json.Marshal(c.State)
		if err != nil {
			log.Fatalln(err)
		}
		m.Respond(jsonReply)
	}); err != nil {
		return err
	}
	return nil
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
				// If PanicTransfer is activated, transfer to vault
				if c.State.Inventory.PanicTransfer.Enabled {
					c.PanicTransfer()
				}
				tmpState := c.State
				if c.State.Inventory.ScanScrambler.Enabled {
					// Use method to generate a random unix timestamp in the future
					c.ScanScramble()
					tmpState.CoolDown.ExpirationTimeStamp = c.ScrambledTimeStamp
				}
				// // gpt71 and Josh countermeasure
				// if foreignCommandCenter.ID == "180887" || foreignCommandCenter.ID == "91548" {
				// 	//tmpState.CoolDown.Time += time.Duration(time.Duration(rand.Intn(10)*100 + 10).Seconds())
				// 	tmpState.CoolDown.ExpirationTimeStamp -= rand.Int63n(100)
				// }

				jsonReply, err := json.Marshal(tmpState) //json.Marshal(c.State)
				if err != nil {
					log.Fatalln(err)
				}
				m.Respond(jsonReply)
			} else {
				// Level is lower. Reply but with empty data
				emptyState := State{ID: c.ID, Nick: c.Nick}
				jsonReply, err := json.Marshal(emptyState)
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
	c.State.Vault.Lock()
	defer c.State.Vault.Unlock()
	if cost > c.State.Vault.Amount {
		err := fmt.Errorf("not enough funds. cost: %f", cost)
		return nil, fmt.Sprintf("Not enough funds. Cost: %f", cost), err
	} else {
		c.State.Vault.Amount = c.State.Vault.Amount - cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(cost))
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
	// QueueSubscribe as sole listener on topic. This way requests will be sequentially worked through.
	if _, err := nc.QueueSubscribe(fmt.Sprintf("commandcenter.%s.stealEndpoint", c.ID), c.ID, func(m *nats.Msg) {
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
					Attacker: struct {
						ID   string "json:\"id\""
						Nick string "json:\"nick\""
					}{
						ID:   foreignCommandCenter.ID,
						Nick: foreignCommandCenter.Nick,
					},
					Defender: struct {
						ID   string "json:\"id\""
						Nick string "json:\"nick\""
					}{
						ID:   c.ID,
						Nick: c.Nick,
					},
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
					Attacker: struct {
						ID   string "json:\"id\""
						Nick string "json:\"nick\""
					}{
						ID:   foreignCommandCenter.ID,
						Nick: foreignCommandCenter.Nick,
					},
					Defender: struct {
						ID   string "json:\"id\""
						Nick string "json:\"nick\""
					}{
						ID:   c.ID,
						Nick: c.Nick,
					},
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

				// attackCooldown := 5 + lvldiff
				// c.StealData.AttackInterval = time.Minute * time.Duration(attackCooldown)
				// Set the cooldown for new steals. Capped at 24 hours max.
				// UPDATE: 2024-04-30: Set to a fixed 10 minutes to allow for a more dynamic gameplay
				c.SetCooldown()
				// Level of stealer is higher

				// Current Funds = 5.11
				// futureFunds   = 5
				// futureFunds   < 5.11? remove 0.11 from Funds and add to stealAttempts, update current Funds
				// Do not remove targetfunds if firewall level changes mid op.
				// Do not remove targetfunds if that would mean sinking below 0

				// Create a fixed size array based on the level difference
				stealAttempts := make([]int, lvldiff)
				// Create a coin cache
				var coincache float32
				// Dyncamically determine amount to steal per attempt by level.
				stealAmount := float32(0.011) * stealerLevel
				log.Printf("Coins are being stolen! %f per attempt!", stealAmount)
				// Try to steal money on each iteration. Logic is described on top.
				c.State.Funds.Lock()
				for range stealAttempts {
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
				// Add stolen amount to personal steal archival list
				c.UpdateStealList(foreignCommandCenter.Nick, coincache)
				monitor.LostCoins.WithLabelValues(c.ID, c.Nick, foreignCommandCenter.ID, foreignCommandCenter.Nick, foreignCommandCenter.Team).Add(float64(coincache))
				// After successfully stealing from the target, send a reply to the attacker
				stealReply := StealReply{
					Attacker: struct {
						ID   string "json:\"id\""
						Nick string "json:\"nick\""
					}{
						ID:   foreignCommandCenter.ID,
						Nick: foreignCommandCenter.Nick,
					},
					Defender: struct {
						ID   string "json:\"id\""
						Nick string "json:\"nick\""
					}{
						ID:   c.ID,
						Nick: c.Nick,
					},
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
	c.State.Vault.Lock()
	c.State.Funds.Lock()
	defer c.State.Vault.Unlock()
	defer c.State.Funds.Unlock()
	// This costs money, so we remove coins based on the level of the stealer. First check if enough funds are available for scan.
	cost := c.State.Stealer.Level * 0.1
	if cost > c.State.Vault.Amount {
		err := fmt.Errorf("not enough funds. cost: %f", cost)
		return stealReply, fmt.Sprintf("Not enough funds. Cost: %f", cost), err
	} else {
		c.State.Vault.Amount = c.State.Vault.Amount - cost
		monitor.SpentCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(cost))
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
	monitor.StolenCoins.WithLabelValues(c.ID, c.Nick, c.Team).Add(float64(stealReply.GainedCoins))
	// Return reply with error on cooldown
	if stealReply.CoolDown {
		err := fmt.Errorf("target is on cooldown")
		return stealReply, "Target is on cooldown", err
	}

	return stealReply, "successful steal operation", nil
}

// Get state of CommandCenter
func (c *CommandCenter) GetState() *CommandCenter {
	return c
}
