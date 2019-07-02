package main

import (
	"fmt"
	"time"
)
import p2p "github.com/wimblechain/wimble-node/p2p"

func main() {
	fmt.Println("WimbleChain v0.0.1a")
	p2p.Start()
	should_exit := false
	const pps = 0.1
	for should_exit == false {
		time.Sleep(1000.0 / pps * time.Millisecond)
		fmt.Println("Tick...")
	}
}
