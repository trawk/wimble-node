package p2p

import (
	"bufio"
	"context"
	"crypto/rand"
	// "flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mrand "math/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"

	golog "github.com/ipfs/go-log"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"
)

// P2P Constants
const Version = "0.0.1a"
const delim = "__DELIM__"
const ConnectAnnounce = "ConnectAnnounce"
const port int = 8383

const STREAM_HANDLER_PROTOCOL_NAME = "/echo/1.0.0"

// Holds on to reference of host, move to instance member for P2P
var ha host.Host

func Start() {
	fmt.Printf("Launching p2p version %s...\n", Version)
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	addr := listen()
	fmt.Println("Launched listener at address: ", addr)

	connect(addr)
}

func listen() string {
	fmt.Println("Open P2P listeniner on port ", port)
	// Make a host that listens on the given multiaddress
	//ha, err := makeBasicHost(port, *insecure, *seed)
	var rand_seed int64 = 0 // TODO: make random
	har, err := makeBasicHost(port, true, rand_seed)
	if err != nil {
		log.Fatal(err)
	}
	ha = har

	ha.SetStreamHandler(STREAM_HANDLER_PROTOCOL_NAME, func(s network.Stream) {
		log.Println("Got a new stream!")
		if err := doEcho(s); err != nil {
			log.Println(err)
			s.Reset()
		} else {
			s.Close()
		}
	})

	log.Println("Listening for connections on port ", port)
	//select {} // hang forever

	// Return addr
	return fmt.Sprintf("%s", getHostAddr(ha))
}

func connect(target string) {
	/*
			listenF := flag.Int("l", 0, "wait for incoming connections")
		target := flag.String("d", "", "target peer to dial")
		insecure := flag.Bool("insecure", false, "use an unencrypted connection")
		seed := flag.Int64("seed", 0, "set random seed for id generation")
		flag.Parse()*/
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(target)
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add it to the peerstore
	// so LibP2P knows how to contact it
	ha.Peerstore().AddAddr(peerid, targetAddr, peerstore.PermanentAddrTTL)

	log.Println("opening stream")
	// make a new stream from host B to host A
	// it should be handled on host A by the handler we set above because
	// we use the same /echo/1.0.0 protocol
	s, err := ha.NewStream(context.Background(), peerid, STREAM_HANDLER_PROTOCOL_NAME)
	if err != nil {
		fmt.Println("Stream error")
		log.Fatalln(err)
	}

	_, err = s.Write([]byte("Hello, world!\n"))
	if err != nil {
		fmt.Println("Write error")
		log.Fatalln(err)
	}

	out, err := ioutil.ReadAll(s)
	if err != nil {
		fmt.Println("Read error")
		log.Fatalln(err)
	}

	log.Printf("read reply: %q\n", out)
}

// makeBasicHost creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It won't encrypt the connection if insecure is true.
func makeBasicHost(listenPort int, insecure bool, randseed int64) (host.Host, error) {

	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	var r io.Reader
	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	if insecure {
		opts = append(opts, libp2p.NoSecurity)
	}

	basicHost, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	log.Printf("I am %s\n", fullAddr)
	if insecure {
		log.Printf("Now run \"./echo -l %d -d %s -insecure\" on a different terminal\n", listenPort+1, fullAddr)
	} else {
		log.Printf("Now run \"./echo -l %d -d %s\" on a different terminal\n", listenPort+1, fullAddr)
	}

	return basicHost, nil
}

func getHostAddr(basicHost host.Host) ma.Multiaddr {
	// Build host multiaddress
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", basicHost.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := basicHost.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	return fullAddr
}

// doEcho reads a line of data a stream and writes it back
func doEcho(s network.Stream) error {
	buf := bufio.NewReader(s)
	str, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	log.Printf("read: %s\n", str)
	_, err = s.Write([]byte(str))
	return err
}
