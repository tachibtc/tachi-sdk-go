package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/tachibtc/daemon-go-sdk/tachi"
)

func main() {
	c, err := tachi.NewClient()
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := c.WS.Subscribe(ctx, tachi.SubscribeOptions{Blocks: true})
	if err != nil {
		fmt.Println("subscribe error:", err)
		return
	}
	defer conn.Close()
	fmt.Println("connected, waiting for a block event...")

	select {
	case evt := <-conn.Events():
		fmt.Printf("got event: %+v\n", evt)
	case err := <-conn.Err():
		fmt.Println("read error:", err)
	case <-time.After(8 * time.Second):
		fmt.Println("timed out waiting for an event (connection itself was fine)")
	}
}
