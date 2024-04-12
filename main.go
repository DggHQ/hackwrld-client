package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	nats "github.com/nats-io/nats.go"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// type AttackerData struct {
// 	Attacker     string  `json:"attacker"`
// 	StealerLevel float32 `json:"stealerlevel"`
// }

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
)

// Handle state endpoint
func state(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"state": commandCenter.State})
}

// Handle miner upgrade endpoint
func minerupgrade(c *gin.Context) {
	success, commandCenter, reply, err := commandCenter.UpgradeCryptoMiner(nc)
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
	success, commandCenter, reply, err := commandCenter.UpgradeScanner(nc)
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
	success, commandCenter, reply, err := commandCenter.UpgradeFirewall(nc)
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
	success, commandCenter, reply, err := commandCenter.UpgradeStealer(nc)
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

// // Generate random floating numbers
// func randFloats(min, max float32, n int) []float32 {
// 	res := make([]float32, n)
// 	for i := range res {
// 		res[i] = min + rand.Float32()*(max-min)
// 	}
// 	return res
// }

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

// // NEEDS TWEAKING: This will handle incoming attacks. Maybe use a simple Pseudo Random Distribution?
// // Example: 100 Numbers between 0 and 1, make a certain amount of them 1, others 0. Shuffle them, if 30 1s are there 30% to hit a 1.
// // https://stackoverflow.com/questions/33994677/pick-a-random-value-from-a-go-slice
// func incomingattack(c *gin.Context) {
// 	var data AttackerData
// 	commandCenter := commandCenter.GetState()
// 	if err := c.ShouldBindJSON(&data); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	// Viable solution:
// 	log.Println(randFloats(data.StealerLevel/commandCenter.State.Firewall.Level, commandCenter.State.Firewall.Level-data.StealerLevel, 1)[0])
// 	random := rand.Float32() * (commandCenter.State.Firewall.Level - data.StealerLevel)
// 	switch chance := data.StealerLevel / commandCenter.State.Firewall.Level; {
// 	case chance >= 1:
// 		log.Printf("Chance: %f - Random: %f", chance, random)
// 		c.JSON(http.StatusOK, gin.H{"success": true})
// 		return
// 	case chance < random:
// 		log.Printf("Chance: %f - Random: %f", chance, random)
// 		c.JSON(http.StatusForbidden, gin.H{"success": false})
// 		return
// 	default:
// 		log.Printf("Chance: %f - Random: %f", chance, random)
// 		c.JSON(http.StatusForbidden, gin.H{"success": true})
// 		return
// 	}
// }

//TODO: Subscribe to game master to get game settings

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

	router := gin.Default()
	router.POST("/upgrade/miner", minerupgrade)
	router.POST("/upgrade/scanner", scannerupgrade)
	router.POST("/upgrade/firewall", firewallupgrade)
	router.POST("/upgrade/stealer", stealerupgrade)
	//router.POST("/attack/in", incomingattack)
	router.POST("/attack/out", func(ctx *gin.Context) {})
	router.POST("/scan/out", scanout)
	router.POST("/steal", steal)
	router.GET("/state", state)

	router.Run(fmt.Sprintf("0.0.0.0:%s", getEnv("PORT", "8088")))
	wg.Wait()

}
