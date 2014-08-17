package main

import (
	peer "github.com/jbenet/go-ipfs/peer"
	dht "github.com/jbenet/go-ipfs/routing/dht"
	"github.com/jbenet/go-ipfs/swarm"
	u "github.com/jbenet/go-ipfs/util"
	ma "github.com/jbenet/go-multiaddr"

	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"net"
	"runtime"
	"time"
)

type dhtInfo struct {
	dht  *dht.IpfsDHT
	addr *ma.Multiaddr
	p    *peer.Peer
}

func _randPeerID() peer.ID {
	buf := make([]byte, 16)
	rand.Read(buf)
	return peer.ID(buf)
}

func _randString() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func setupDHT(addr string) *dhtInfo {
	addr_ma, err := ma.NewMultiaddr(addr)
	if err != nil {
		panic(err)
	}

	peer_dht := new(peer.Peer)
	peer_dht.AddAddress(addr_ma)
	peer_dht.ID = _randPeerID()

	net := swarm.NewSwarm(peer_dht)
	err = net.Listen()
	if err != nil {
		panic(err)
	}

	ndht := dht.NewDHT(peer_dht, net)

	ndht.Start()
	return &dhtInfo{ndht, addr_ma, peer_dht}
}

func ConnectPeers(dhts []*dhtInfo, a, b int) {
	di_a := dhts[a]
	di_b := dhts[b]

	_, err := di_a.dht.Connect(di_b.addr)
	if err != nil {
		panic(err)
	}
}

func PingBetween(dhts []*dhtInfo, a, b int) {
	dhts[a].dht.Ping(dhts[b].p, time.Second*2)
}

const (
	CONNECT = iota
	PING
	GET
	PUT
	PROVIDE
	FINDPROVIDE
)

var dhts []*dhtInfo
var keys chan u.Key
var provs chan u.Key

func init() {
	keys = make(chan u.Key, 1000)
	provs = make(chan u.Key, 1000)
}

func hailMaryDHT(dh *dhtInfo) {
	var mycons []*peer.Peer
	for i := 0; i < 5; i++ {
		o_id := mrand.Intn(len(dhts))
		if dh == dhts[o_id] {
			i--
			continue
		}
		_, err := dh.dht.Connect(dhts[o_id].addr)
		if err != nil {
			panic(err)
		}
		mycons = append(mycons, dhts[o_id].p)
	}

	fmt.Println("DHT done with connects.")
	for {
		a := mrand.Intn(5) + 1
		switch a {
		case PING:
			fmt.Println("ACTION: ping")
			o_id := mrand.Intn(len(mycons))
			err := dh.dht.Ping(mycons[o_id], time.Second*2)
			if err != nil {
				fmt.Println("Ping failed...")
			}
			fmt.Println("ACTION: ping finished")
		case PUT:
			fmt.Println("ACTION: put")
			k := u.Key(_randString())
			dh.dht.PutValue(k, []byte(_randString()))
			keys <- k
			fmt.Println("ACTION: put finished")
		case GET:
			fmt.Println("ACTION: get")
			k, ok := <-keys
			if !ok {
				fmt.Println("ACTION: get continued")
				continue
			}
			fmt.Println("ACTION: get key")
			_, err := dh.dht.GetValue(k, time.Second*2)
			if err != nil {
				if err == u.ErrSearchIncomplete {
					fmt.Println("Didnt find value on first try...")
				} else if err == u.ErrNotFound {
					fmt.Println("Failed to find value at all.")
				} else if err == u.ErrTimeout {
					fmt.Println("CAUTION: Call timed out!!")
				} else {
					panic(err)
				}
			}
			keys <- k
			fmt.Println("ACTION: get finished")
		case PROVIDE:
			fmt.Println("ACTION: provide")
			k := u.Key(_randString())
			err := dh.dht.Provide(k)
			if err != nil {
				panic(err)
			}
			provs <- k
			fmt.Println("ACTION: provide finished")
		case FINDPROVIDE:
			fmt.Println("ACTION: find provider")
			k := <-provs
			_, err := dh.dht.FindProviders(k, time.Second*2)
			if err != nil {
				if err == u.ErrNotFound {
					fmt.Println("Couldnt find provider.")
				} else {
					panic(err)
				}
			}
			provs <- k

			fmt.Println("ACTION: find provider finished")

		}
	}
}

func main() {
	u.Debug = true
	runtime.GOMAXPROCS(10)

	go func() { //If you need to stop the simulation and inspect all goroutines
		list, err := net.Listen("tcp", ":4999")
		if err != nil {
			panic(err)
		}

		list.Accept()
		panic("Lets take a look at things.")
	}()
	for i := 0; i < 50; i++ {
		dhts = append(dhts, setupDHT(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 5000+i)))
	}
	fmt.Println("Finished DHT creation.")

	for _, d := range dhts {
		go hailMaryDHT(d)
	}

	fmt.Println("Finished start test.")
	for {
		time.Sleep(time.Hour)
	}

	return
}
