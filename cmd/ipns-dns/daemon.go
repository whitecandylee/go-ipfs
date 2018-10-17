package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	routing "gx/ipfs/QmPmFeQ5oY5G6M7aBWggi5phxEPXwsQntE1DFcUzETULdp/go-libp2p-routing"
	blocks "gx/ipfs/QmRcHuYzAyswytBuMF78rj3LTChYszomRFXNg4685ZN1WM/go-block-format"
	record "gx/ipfs/QmSb4B8ZAAj5ALe9LjfzPyF8Ma6ezC1NTnDF2JQPUJxEXb/go-libp2p-record"
	dht "gx/ipfs/QmSteomMgXnSQxLEY5UpxmkYAd8QF9JuLLeLYBokTHxFru/go-libp2p-kad-dht"
	dhtopts "gx/ipfs/QmSteomMgXnSQxLEY5UpxmkYAd8QF9JuLLeLYBokTHxFru/go-libp2p-kad-dht/opts"
	floodsub "gx/ipfs/QmTcC9Qx2adsdGguNpqZ6dJK7MMsH8sf3yfxZxG3bSwKet/go-libp2p-floodsub"
	peerstore "gx/ipfs/QmWtCpWB39Rzc2xTB75MKorsxNpo3TyecTEN24CJ3KVohE/go-libp2p-peerstore"
	ipns "gx/ipfs/QmX72XT6sSQRkNHKcAFLM2VqB3B4bWPetgWnHY8LgsUVeT/go-ipns"
	ipnspb "gx/ipfs/QmX72XT6sSQRkNHKcAFLM2VqB3B4bWPetgWnHY8LgsUVeT/go-ipns/pb"
	maddr "gx/ipfs/QmYmsdtJ3HsodkePE3eU3TsCaP2YvPZJ4LoXnNkDE5Tpt7/go-multiaddr"
	datastore "gx/ipfs/QmaRb5yNXKonhbkpNxNawoydk4N6es6b4fPj19sjEKsh5D/go-datastore"
	p2ppeer "gx/ipfs/QmbNepETomvmXfz1X5pHNFD2QuPqnqi47dTd94QJWSorQ3/go-libp2p-peer"
	p2phost "gx/ipfs/Qmf5yHzmWAyHSJRPAmZzfk3Yd7icydBLi7eec5741aov7v/go-libp2p-host"
)

type Daemon struct {
	Host    p2phost.Host
	Routing routing.IpfsRouting
	PubSub  *floodsub.PubSub
	Updates *floodsub.Subscription

	Records map[p2ppeer.ID]*ipnspb.IpnsEntry
}

func NewDaemon(ctx context.Context, host p2phost.Host) (*Daemon, error) {
	d := &Daemon{Host: host}

	pubsub, err := floodsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}

	ds := datastore.NewMapDatastore()
	validator := record.NamespacedValidator{
		"pk":   record.PublicKeyValidator{},
		"ipns": ipns.Validator{KeyBook: host.Peerstore()},
	}
	dht, err := dht.New(ctx, host, dhtopts.Datastore(ds), dhtopts.Validator(validator))
	if err != nil {
		return nil, err
	}

	d.PubSub = pubsub
	d.Routing = dht

	return d, nil
}

func (d *Daemon) Bootstrap(ctx context.Context, addrs []string, topic string) error {
	if err := d.BootstrapNetwork(ctx, addrs); err != nil {
		return err
	}
	return d.Subscribe(topic)
}

func (d *Daemon) BootstrapNetwork(ctx context.Context, addrs []string) error {
	for _, a := range addrs {
		pinfo, err := peerstore.InfoFromP2pAddr(maddr.StringCast(a))
		if err != nil {
			return err
		}
		if err = d.Host.Connect(ctx, *pinfo); err != nil {
			return err
		}
		fmt.Printf("connected: /p2p/%s\n", pinfo.ID.Pretty())
	}

	return nil
}

// see also: go-libp2p-pubsub-router/pubsub.go
//
// TODO: keep looking for providers
func (d *Daemon) AnnouncePubsub(ctx context.Context, topic string) error {
	timeout := 120 * time.Second

	cid := blocks.NewBlock([]byte("floodsub:" + topic)).Cid()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := d.Routing.Provide(ctx, cid, true); err != nil {
		return err
	}

	return nil
}

func (d *Daemon) MaintainPubsub(ctx context.Context, topic string) error {
	searchMax := 10
	searchTimeout := 30 * time.Second
	connectTimeout := 10 * time.Second

	cid := blocks.NewBlock([]byte("floodsub:" + topic)).Cid()

	sctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()

	provs := d.Routing.FindProvidersAsync(sctx, cid, searchMax)
	wg := &sync.WaitGroup{}
	for p := range provs {
		wg.Add(1)
		go func(pi peerstore.PeerInfo) {
			defer wg.Done()
			ctx, cancel2 := context.WithTimeout(ctx, connectTimeout)
			defer cancel2()
			err := d.Host.Connect(ctx, pi)
			if err != nil {
				return
			}
		}(p)
	}
	wg.Wait()

	return nil
}

func (d *Daemon) Subscribe(topic string) error {
	if err := d.PubSub.RegisterTopicValidator(topic, d.validateMessage); err != nil {
		return err
	}

	sub, err := d.PubSub.Subscribe(topic)
	if err != nil {
		return err
	}

	d.Updates = sub
	return nil
}

func (d *Daemon) validateMessage(ctx context.Context, msg *floodsub.Message) bool {
	return true
}

func (d *Daemon) ReceiveUpdates(ctx context.Context) {
	for {
		msg, err := d.Updates.Next(ctx)
		if err != nil {
			continue
		}
		fmt.Printf("received: %s (seqno %d)\n", msg.Data, msg.Seqno)
	}
}

func (d *Daemon) ServeDNS(ctx context.Context)  {}
func (d *Daemon) ServeHTTP(ctx context.Context) {}
