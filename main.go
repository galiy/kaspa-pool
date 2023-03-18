package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/galiy/kaspa-pool/src/kaspastratum"
	"gopkg.in/yaml.v2"
)

func main() {
	pwd, _ := os.Getwd()
	fullPath := path.Join(pwd, "config.yaml")
	log.Printf("loading config @ `%s`", fullPath)
	rawCfg, err := ioutil.ReadFile(fullPath)
	if err != nil {
		log.Printf("config file not found: %s", err)
		os.Exit(1)
	}

	cfg := kaspastratum.BridgeConfig{}

	if err := yaml.Unmarshal(rawCfg, &cfg); err != nil {
		log.Printf("failed parsing config file: %s", err)
		os.Exit(1)
	}

	flag.StringVar(&cfg.StratumPort, "stratum", cfg.StratumPort, "stratum port to listen on, default `:5555`")
	flag.StringVar(&cfg.RPCServer, "kaspa", cfg.RPCServer, "address of the kaspad node, default `localhost:16110`")
	flag.DurationVar(&cfg.BlockWaitTime, "blockwait", cfg.BlockWaitTime, "time in ms to wait before manually requesting new block, default `500`")
	flag.UintVar(&cfg.MinShareDiff, "mindiff", cfg.MinShareDiff, "minimum share difficulty to accept from miner(s), default `4`")
	flag.UintVar(&cfg.ExtranonceSize, "extranonce", cfg.ExtranonceSize, "size in bytes of extranonce, default `0`")
	flag.BoolVar(&cfg.UseLogFile, "log", cfg.UseLogFile, "if true will output errors to log file, default `true`")
	flag.StringVar(&cfg.PoolWallet, "pwallet", cfg.PoolWallet, `kaspa wallet for pool use. If empty - set to client wallet`)
	flag.StringVar(&cfg.OraConnStr, "pconnect", cfg.OraConnStr, `oracle connect string. sample: user="oraUserName" password="oraPassword" connectString="tnsName" noTimezoneCheck=true`)
	flag.Parse()

	if cfg.MinShareDiff == 0 {
		cfg.MinShareDiff = 4
	}
	if cfg.BlockWaitTime == 0 {
		cfg.BlockWaitTime = 5 * time.Second // this should never happen due to kas 1s block times
	}

	log.Println("----------------------------------")
	log.Printf("initializing bridge")
	log.Printf("\tkaspad:          %s", cfg.RPCServer)
	log.Printf("\tstratum:         %s", cfg.StratumPort)
	log.Printf("\tlog:             %t", cfg.UseLogFile)
	log.Printf("\tmin diff:        %d", cfg.MinShareDiff)
	log.Printf("\tblock wait:      %s", cfg.BlockWaitTime)
	log.Printf("\textranonce size: %d", cfg.ExtranonceSize)
	log.Printf("\tpool wallet:     %s", cfg.PoolWallet)
	re := regexp.MustCompile(`password=".*?"`)
	ocsWrap := re.ReplaceAllString(cfg.OraConnStr, `password="***"`)
	log.Printf("\toracle connect:  %s", ocsWrap)

	log.Println("----------------------------------")

	if err := kaspastratum.ListenAndServe(cfg); err != nil {
		log.Println(err)
	}
}
