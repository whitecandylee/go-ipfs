package dnspubsub

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	namesys "github.com/ipfs/go-ipfs/namesys"
	namesysopt "github.com/ipfs/go-ipfs/namesys/opts"

	cid "gx/ipfs/QmPSQnBKM9g7BaUcZCvswUJVscQ1ipjmwxN5PXCjkp9EQ7/go-cid"
	multihash "gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	p2pcrypto "gx/ipfs/QmPvyPwuCgJ7pDmrKDxRtsScJgBaM5h4EpRL2qQJsmXf4n/go-libp2p-crypto"
	// dns "gx/ipfs/QmWchsfMt9Re1CQaiHqPQC1DrZ9bkpa6n229dRYkGyLXNh/dns"
	floodsub "gx/ipfs/QmTcC9Qx2adsdGguNpqZ6dJK7MMsH8sf3yfxZxG3bSwKet/go-libp2p-floodsub"
	ipns "gx/ipfs/QmX72XT6sSQRkNHKcAFLM2VqB3B4bWPetgWnHY8LgsUVeT/go-ipns"
	ipnspb "gx/ipfs/QmX72XT6sSQRkNHKcAFLM2VqB3B4bWPetgWnHY8LgsUVeT/go-ipns/pb"
	path "gx/ipfs/QmdrpbDgeYH3VxkCciQCJY5LkDYdXtig6unDzQmMxFtWEw/go-path"
	proto "gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
	multibase "gx/ipfs/QmekxXDhCxCJRNuzmHreuaT3BsuJcsjcXWNrtV9C8DRHtd/go-multibase"
)

type Namesys struct {
	PubSub *floodsub.PubSub
	DNS    *net.Resolver
}

func NewNamesys(pubsub *floodsub.PubSub, resolver *net.Resolver) Namesys {
	return Namesys{pubsub, resolver}
}

func (n Namesys) Resolve(ctx context.Context, namepath string, opts ...namesysopt.ResolveOpt) (path.Path, error) {
	if !strings.HasPrefix(namepath, "/ipns/") {
		return "", fmt.Errorf("not an ipns name: %s", namepath)
	}

	peerid, err := multihash.Cast(strings.Split(namepath, "/")[2])
	if err != nil {
		return "", fmt.Errorf("failed to decode PeerID: %s", err)
	}
	peercid := NewCidV1(cotext)

	peeridb32 := peerid.Encode(multibase.MustNewEncoder('b'))

	records, err := n.DNS.LookupTXT(ctx, peeridb32+".ipns.name")
	if err != nil {
		return "", err
	}

	var bestPath path.Path
	var bestEOL time.Time
	for _, str := range records {
		if !strings.HasPrefix(str, "ipns=") {
			continue
		}
		_, pb, err := multibase.Decode(str)
		if err != nil {
			continue
		}

		entry := new(ipnspb.IpnsEntry)
		err = proto.Unmarshal(pb, entry)
		if err != nil {
			continue
		}
		p, err := path.ParsePath(string(entry.GetValue()))
		if err != nil {
			continue
		}

		eol, err := ipns.GetEOL(entry)
		if err != nil {
			continue
		}
		if eol.Sub(bestEOL) < 0 {
			continue
		}

		bestPath, bestEOL = p, eol
	}

	if len(bestPath) == 0 {
		return "", namesys.ErrResolveFailed
	}

	return bestPath, nil
}

func (n Namesys) Publish(ctx context.Context, name p2pcrypto.PrivKey, value path.Path) error {
	arbitraryEOL := time.Now().Add(24 * time.Hour)
	return n.PublishWithEOL(ctx, name, value, arbitraryEOL)
}

// pubsub
// ipns entry as base64
func (n Namesys) PublishWithEOL(ctx context.Context, name p2pcrypto.PrivKey, value path.Path, eol time.Time) error {
	return nil
}
