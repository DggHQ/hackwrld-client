package main

import (
	"log"
	"sync"
	"time"

	nats "github.com/nats-io/nats.go"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	//defaultConfigFile = "config.json"
	baseMineRate = 0.001
)

var (
	// config        CommandCenter
	commandCenter CommandCenter
	nc, natsError = nats.Connect("localhost", nil, nats.PingInterval(20*time.Second), nats.MaxPingsOutstanding(5))
	clientConfig  = clientv3.Config{
		Endpoints:   []string{"10.10.90.5:2379", "10.10.90.6:2379"},
		DialTimeout: time.Second * 5,
	}
)

//TODO: Subscribe to game master to get game settings

func main() {
	if natsError != nil {
		log.Fatalln(natsError)
	}
	defer nc.Close()
	commandCenter = commandCenter.Init(clientConfig)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	// // Config load
	// file := defaultConfigFile
	// f, err := ioutil.ReadFile(file)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// err = json.Unmarshal(f, &config)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// // end of config load

	// commandCenter = commandCenter.Prepare(config)

	go commandCenter.Mine(wg)

	go func() {
		time.Sleep(time.Second * 5)
		_, err := commandCenter.UpgradeCryptoMiner(nc)
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// if natsError != nil {
	// 	log.Fatalln(natsError)
	// }
	// defer nc.Close()
	// go commandCenter.PublishState(nc, wg) //publishState(wg)
	wg.Wait()

}
