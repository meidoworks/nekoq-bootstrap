package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/google/gops/agent"

	bootstrap "github.com/meidoworks/nekoq-bootstrap"
)

type Config struct {
	Main struct {
		StorageProvider string `toml:"storage_provider"`
		Debug           bool   `toml:"debug"`
	} `toml:"main"`
	Cluster struct {
		ClusterName   string            `toml:"cluster_name"`
		ClusterSecret string            `toml:"cluster_secret"`
		NodeName      string            `toml:"node_name"`
		Listener      string            `toml:"listener"`
		Nodes         map[string]string `toml:"nodes"`
	} `toml:"cluster"`
	Dns struct {
		Enable      bool   `toml:"enable"`
		Address     string `toml:"listener"`
		HttpAddress string `toml:"http_listener"`
		StaticRules struct {
			A map[string]string `toml:"A"`
		} `toml:"static_rule"`
	} `toml:"dns"`
	Http struct {
		Listener       string `toml:"listener"`
		EnableAuth     bool   `toml:"enable_auth"`
		AccessPassword string `toml:"access_password"`
	} `toml:"http"`
}

func main() {
	// init gops
	if err := agent.Listen(agent.Options{}); err != nil {
		log.Fatal(err)
	}

	config := new(Config)
	m, err := toml.DecodeFile("bootstrap.toml", config)
	if err != nil {
		panic(err)
	}
	var _ = m

	var storage bootstrap.Storage
	switch config.Main.StorageProvider {
	case "mem":
		storage = bootstrap.NewMemStore()
	default:
		panic(errors.New("unknown storage provider"))
	}

	// cluster
	{
		ha, err := bootstrap.NewHaModule(
			config.Cluster.NodeName,
			config.Cluster.Listener,
			config.Cluster.ClusterName,
			config.Cluster.ClusterSecret,
			config.Cluster.Nodes,
			storage,
		)
		if err != nil {
			panic(err)
		}
		if config.Main.Debug {
			ha.DebugPrint = true
		}

		//TODO deferred
		go func() {
			err := ha.StartSync()
			if err != nil {
				panic(err)
			}
		}()
	}

	// dns
	if config.Dns.Enable {
		for k, v := range config.Dns.StaticRules.A {
			storage.PutDomain(k, v, bootstrap.DomainTypeA)
		}

		endpoint, err := bootstrap.NewDnsEndpoint(config.Dns.Address, storage)
		if err != nil {
			panic(err)
		}
		httpEndpoint, err := bootstrap.NewHttpDns(config.Dns.HttpAddress, storage)
		if err != nil {
			panic(err)
		}
		if config.Main.Debug {
			endpoint.DebugPrintDnsRequest = true
			httpEndpoint.DebugPrintDnsRequest = true
		}

		log.Println("[INFO] start dns module at", config.Dns.Address)
		log.Println("[INFO] start dns-http module at", config.Dns.HttpAddress)

		//TODO deferred
		go func() {
			err := endpoint.StartSync()
			if err != nil {
				panic(err)
			}
		}()
		go func() {
			err := httpEndpoint.StartSync()
			if err != nil {
				panic(err)
			}
		}()
	}

	// http
	{
		httpEP, err := bootstrap.NewHttpEndpoint(config.Http.Listener, storage, config.Http.EnableAuth, config.Http.AccessPassword)
		if err != nil {
			panic(err)
		}

		//TODO deferred
		go func() {
			err := httpEP.StartSync()
			if err != nil {
				panic(err)
			}
		}()
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	fmt.Println("signal received:", sig)
}
