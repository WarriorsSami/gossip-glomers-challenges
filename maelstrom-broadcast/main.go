package main

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastBody struct {
	maelstrom.MessageBody
	Message int `json:"message"`
}

type ReadBody struct {
	Type     string `json:"type"`
	Messages []int  `json:"messages"`
}

type TopologyBody struct {
	maelstrom.MessageBody
	Topology map[string][]string `json:"topology"`
}

type ServerNode struct {
	*maelstrom.Node
	sync.Mutex
	Messages  map[int]bool
	Neighbors []string
}

func NewServerNode() *ServerNode {
	return &ServerNode{
		Node:      maelstrom.NewNode(),
		Messages:  make(map[int]bool),
		Neighbors: make([]string, 0),
	}
}

func main() {
	n := NewServerNode()

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		_, seen := n.Messages[body.Message]
		n.Messages[body.Message] = true
		n.Unlock()

		if !seen {
			for _, neighbor := range n.Neighbors {
				n.Send(neighbor, body)
			}
		}

		return n.Reply(msg, maelstrom.MessageBody{Type: "broadcast_ok"})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body maelstrom.MessageBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		msgs := make([]int, 0, len(n.Messages))
		for msgVal := range n.Messages {
			msgs = append(msgs, msgVal)
		}
		n.Unlock()

		resp := ReadBody{
			Type:     "read_ok",
			Messages: msgs,
		}

		return n.Reply(msg, resp)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		var body TopologyBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		n.Neighbors = body.Topology[n.ID()]
		n.Unlock()

		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
