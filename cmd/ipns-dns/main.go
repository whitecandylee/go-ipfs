package main

import (
	"context"
	"fmt"
	"time"

	libp2p "gx/ipfs/QmPL3AKtiaQyYpchZceXBZhZ3MSnoGqJvLZrc7fzDTTQdJ/go-libp2p"
)

const topic = "/ipns/.well-known/all"

var bootstrap = []string{
	// "/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	// "/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	// "/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := libp2p.New(ctx)
	if err != nil {
		panic(err)
	}

	for _, ma := range host.Network().ListenAddresses() {
		fmt.Printf("listening: %s/p2p/%s\n", ma.String(), host.ID().Pretty())
	}

	d, err := NewDaemon(ctx, host)
	if err != nil {
		panic(err)
	}
	if err = d.Bootstrap(ctx, bootstrap, topic); err != nil {
		panic(err)
	}

	fmt.Printf("bootstrapped: ok\n")

	go d.ReceiveUpdates(ctx)
	go d.ServeDNS(ctx)
	go d.ServeHTTP(ctx)

	go func() {
		for {
			fmt.Printf("announcing pubsub...\n")
			d.AnnouncePubsub(ctx, topic)
			fmt.Printf("announcing pubsub: done\n")
			time.Sleep(30 * time.Second)
		}
	}()

	go func() {
		for {
			fmt.Printf("maintaining pubsub...\n")
			d.MaintainPubsub(ctx, topic)
			fmt.Printf("maintaining pubsub: done\n")
			time.Sleep(30 * time.Second)
		}
	}()

	go func() {
		for range time.Tick(10 * time.Second) {
			fmt.Printf("peers: total %d, topic %d\n",
				len(host.Network().Conns()), len(d.PubSub.ListPeers(topic)))
		}
	}()

	select {}
}
