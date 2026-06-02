package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type AddBody struct {
	maelstrom.MessageBody
	Delta int `json:"delta"`
}

type ReadBody struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type ServerNode struct {
	*maelstrom.Node
	*maelstrom.KV
	sync.Mutex
	localCounter int
}

func NewServerNode() *ServerNode {
	node := maelstrom.NewNode()

	return &ServerNode{
		Node: node,
		KV:   maelstrom.NewSeqKV(node),
	}
}

func (n *ServerNode) trySync() error {
	// grab local counter
	n.Lock()
	delta := n.localCounter
	n.Unlock()

	if delta == 0 {
		return nil
	}

	// read last value from the partial global counter
	curr, err := n.ReadInt(context.Background(), n.ID())
	if err != nil {
		curr = 0
	}

	// try to CAS it
	err = n.CompareAndSwap(context.Background(), n.ID(), curr, curr+delta, true)

	if err != nil {
		return err
	}
	
	// clear local counter if successful
	n.Lock()
	n.localCounter -= delta
	n.Unlock()

	return nil
}

func main() {
	n := NewServerNode()

	n.Handle("add", func(msg maelstrom.Message) error {
		var body AddBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// add to local counter
		n.Lock()
		n.localCounter += body.Delta 
		n.Unlock()

		// try to sync with global counter
		n.trySync()

		return n.Reply(msg, maelstrom.MessageBody{Type: "add_ok"})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body maelstrom.MessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
 
		// try to evict the local counter to the global one if possible
		err := n.trySync()

		// sweep over the global partial counters and sum them up
		total := 0
		for _, nodeID := range n.NodeIDs() {
			val, err := n.ReadInt(context.Background(), nodeID)
			if err == nil {
				total += val
			}
		}

		// if not possible, fallback to best effort option and add the local counter on demand
		if err != nil {
			n.Lock()
			total += n.localCounter
			n.Unlock()
		}

		resp := ReadBody{
			Type:  "read_ok",
			Value: total,
		}

		return n.Reply(msg, resp)
	})

	// run a periodic sync goroutine
	go func () {
		for {
			time.Sleep(200 * time.Millisecond)
			n.trySync()
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
