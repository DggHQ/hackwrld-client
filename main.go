package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	nats "github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type StealRequest struct {
	TargetId string `json:"targetid"`
}

const (
	baseMineRate = 0.001
)

var (
	commandCenter CommandCenter
	nc, natsError = nats.Connect(getEnv("NATS_HOST", "localhost"), nil, nats.PingInterval(20*time.Second), nats.MaxPingsOutstanding(5))
	clientConfig  = clientv3.Config{
		Endpoints:   getEnvToArray("ETCD_ENDPOINTS", "10.10.90.5:2379;10.10.90.6:2379"),
		DialTimeout: time.Second * 5,
	}
	monitor = Monitor{}
)

// Handle state endpoint
func state(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/state").Inc()
	c.JSON(http.StatusOK, gin.H{"state": commandCenter.GetState().State})
}

// Handle miner upgrade endpoint
func minerupgrade(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/miner").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeCryptoMiner(nc, false)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle miner upgrade endpoint
func minerupgrademax(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/miner/max").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeCryptoMiner(nc, true)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle scanner upgrade endpoint
func scannerupgrade(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/scanner").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeScanner(nc, false)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle scanner upgrade endpoint
func scannerupgrademax(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/scanner/max").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeScanner(nc, true)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle firewall upgrade endpoint
func firewallupgrade(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/firewall").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeFirewall(nc, false)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle firewall upgrade endpoint
func firewallupgrademax(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/firewall/max").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeFirewall(nc, true)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle stealer upgrade endpoint
func stealerupgrade(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/stealer").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeStealer(nc, false)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle stealer upgrade endpoint
func stealerupgrademax(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/stealer/max").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeStealer(nc, true)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle scan outgoing endpoint
func scanout(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/scan/out").Inc()

	scans, msg, err := commandCenter.RequestScan(nc)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "scans": scans, "message": msg})
}

// Handle storing coins in vault
func storevault(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/vault/store").Inc()

	transferAmount := commandCenter.StoreVault()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("Transferred %f to vault.", transferAmount)})
}

// // Handle withdrawing coins in vault
// func withdrawvault(c *gin.Context) {
// 	transferAmount := commandCenter.WithdrawVault()
// 	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("Withdrew %f from vault.", transferAmount)})
// }

// Handle steal event
func steal(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/steal").Inc()

	var data StealRequest
	// Could not parse body
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "reply": nil, "message": err.Error()})
		return
	}
	// Try stealing from target. Target is determined by body
	reply, msg, err := commandCenter.StealFromTarget(data.TargetId, nc)
	if err != nil {
		// Not enough funds or cooldown
		c.JSON(http.StatusForbidden, gin.H{"success": false, "reply": reply, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "reply": reply, "message": msg})
}

// Handle vault upgrade endpoint
func vaultupgrade(c *gin.Context) {
	monitor.Requests.WithLabelValues(commandCenter.ID, commandCenter.Nick, "/upgrade/vault").Inc()

	success, commandCenter, reply, err := commandCenter.UpgradeVault(nc, false)
	if err != nil {
		log.Fatalln(err)
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown error.", "state": commandCenter.State, "reply": reply})
	}
	if success {
		c.JSON(http.StatusOK, gin.H{"message": "upgraded", "state": commandCenter.State, "reply": reply})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": "not enough funds.", "state": commandCenter.State, "reply": reply})
}

// Handle vault miner endpoint
func vaultmineractivate(c *gin.Context) {
	success, err := commandCenter.ActivateVaultMiner()
	// We take the error message as the reply
	if success {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("v4ultM1ner.exe activated. %f coins will be mined to Vault.", commandCenter.GetState().State.Inventory.VaultMiner.AmountLeft), "state": commandCenter.State})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": err.Error(), "state": commandCenter.State})
}

// Handle panic transfer endpoint
func panictransferactivate(c *gin.Context) {
	success, err := commandCenter.ActivatePanicTransfer()
	// We take the error message as the reply
	if success {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("panictransfer.bin activated. For %.0f enemy scans, your coins will be automatically transferred to the vault for you.", commandCenter.GetState().State.Inventory.PanicTransfer.AmountLeft), "state": commandCenter.State})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": err.Error(), "state": commandCenter.State})
}

// Handle scan scrambler endpoint
func scanscrambleractivate(c *gin.Context) {
	success, err := commandCenter.ActivateScanScrambler()
	// We take the error message as the reply
	if success {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("scanscr4mbl.sys activated. For %.0f enemy scans, fake cooldown times will be sent", commandCenter.GetState().State.Inventory.ScanScrambler.AmountLeft), "state": commandCenter.State})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{"message": err.Error(), "state": commandCenter.State})
}

// Cheat endpoint only available locally and not reachable via webgui
func cheat(c *gin.Context) {
	commandCenter.State.Vault.Lock()
	defer commandCenter.State.Vault.Unlock()
	commandCenter.State.Vault.Amount += 1000
	c.JSON(http.StatusOK, gin.H{"success": "added coins to vault."})
}

//TODO: Subscribe to game master to get game settings

func init() {
	monitor.Init()
}
func promHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

// Run commandCenter after config init
func main() {
	if natsError != nil {
		log.Fatalln(natsError)
	}
	defer nc.Close()
	commandCenter = commandCenter.Init(clientConfig)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	// Mine in background
	go commandCenter.Mine(wg)
	// Listen to scan topic and reply if conditions are met
	go commandCenter.ReplyScan(nc)
	// Send Player info to leaderboard container when queried
	go commandCenter.SendPlayerInfo(nc)
	// Listen to steal events on client
	go commandCenter.ReplySteal(nc)
	// Run task to continuosly save state each minute.
	// This is done to reduce load on the storage backend.
	go commandCenter.SaveStateContinuous()
	// Listen to topic to reset the instance
	go commandCenter.ListenReset(nc)

	router := gin.Default()
	router.POST("/upgrade/miner", minerupgrade)
	router.POST("/upgrade/miner/max", minerupgrademax)
	router.POST("/upgrade/scanner", scannerupgrade)
	router.POST("/upgrade/scanner/max", scannerupgrademax)
	router.POST("/upgrade/firewall", firewallupgrade)
	router.POST("/upgrade/firewall/max", firewallupgrademax)
	router.POST("/upgrade/stealer", stealerupgrade)
	router.POST("/upgrade/stealer/max", stealerupgrademax)
	router.POST("/vault/store", storevault)
	router.POST("/upgrade/vault", vaultupgrade)
	router.POST("/shop/activate/vaultminer", vaultmineractivate)
	router.POST("/shop/activate/panictransfer", panictransferactivate)
	router.POST("/shop/activate/scanscrambler", scanscrambleractivate)
	// router.POST("/vault/withdraw", withdrawvault)
	router.POST("/attack/out", func(ctx *gin.Context) {})
	router.POST("/scan/out", scanout)
	router.POST("/steal", steal)
	router.GET("/state", state)
	router.GET("/cheat", cheat)
	router.GET("/metrics", promHandler())

	router.Run(fmt.Sprintf("0.0.0.0:%s", getEnv("PORT", "8088")))
	wg.Wait()

}
