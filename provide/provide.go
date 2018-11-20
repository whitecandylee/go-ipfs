package provide

import (
	"context"
	"fmt"
	"gx/ipfs/QmPJUtEJsm5YLUWhF6imvyCH8KZXRJa9Wup7FDMwTy5Ufz/backoff"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	"gx/ipfs/QmYMQuypUbgsdNHmuCBSUJV6wdQVsBHRivNAp3efHJwZJD/go-verifcid"
	"gx/ipfs/QmZBH87CAPFHcc7cYmBqeSQ98zQ3SX9KUxiYgzPmLWNVKz/go-libp2p-routing"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

type Strategy func(context.Context, chan cid.Cid, cid.Cid)

type Provider struct {
	ctx context.Context
	outgoing chan cid.Cid
	strategy Strategy

	// temp, maybe:
	contentRouting routing.ContentRouting
}

var (
	log = logging.Logger("provider")
	provideOutgoingLimit = 100
	provideWorkerCount = 6
)

func NewProvider(ctx context.Context, strategy Strategy, contentRouting routing.ContentRouting) *Provider {
	return &Provider{
		ctx: ctx,
		outgoing: make(chan cid.Cid, provideOutgoingLimit),
		strategy: strategy,
		contentRouting: contentRouting,
	}
}

// Start workers to handle provide requests.
func (p *Provider) Run() {
	for i := 0; i < provideWorkerCount; i++ {
		go p.handleOutgoing(p.outgoing)
	}
}

// Provide the given cid using specified strategy.
//
// * Provide occurs asynchronously, with back pressure applied. Calls
//	 to Provide can block (at least right now).
func (p *Provider) Provide(root cid.Cid) {
	p.strategy(p.ctx, p.outgoing, root)
}

// Announce to the world that a block is being provided.
//
// TODO: Refactor duplication between here and the reprovider.
func (p *Provider) Announce(cid cid.Cid) error {
	// hash security
	if err := verifcid.ValidateCid(cid); err != nil {
		log.Errorf("insecure hash in provider, %s (%s)", cid, err)
		return nil
	}

	op := func() error {
		err := p.contentRouting.Provide(p.ctx, cid, true)
		if err != nil {
			log.Debugf("Failed to provide key: %s", err)
		}
		return err
	}

	if err := backoff.Retry(op, backoff.NewExponentialBackOff()); err != nil {
		log.Debugf("Providing failed after number of retries: %s", err)
		return err
	}

	return nil
}

// Workers

func (p *Provider) handleOutgoing(outgoing <-chan cid.Cid) {
	for cid := range outgoing {
		// Provide these things
		fmt.Println("handleOutgoing", cid)
		p.Announce(cid)
		fmt.Println("announced", cid)
	}
}
