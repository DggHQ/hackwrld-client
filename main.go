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
	c.JSON(http.StatusOK, gin.H{"state": commandCenter.GetState().State})
}

// Handle miner upgrade endpoint
func minerupgrade(c *gin.Context) {
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
	scans, msg, err := commandCenter.RequestScan(nc)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "scans": scans, "message": msg})
}

// Handle steal event
func steal(c *gin.Context) {
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
	// Listen to steal events on client
	go commandCenter.ReplySteal(nc)
	// Update cooldown value in goroutine
	go commandCenter.UpdateCoolDown()
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
	router.POST("/attack/out", func(ctx *gin.Context) {})
	router.POST("/scan/out", scanout)
	router.POST("/steal", steal)
	router.GET("/state", state)
	router.GET("/metrics", promHandler())

	router.Run(fmt.Sprintf("0.0.0.0:%s", getEnv("PORT", "8088")))
	wg.Wait()

}
