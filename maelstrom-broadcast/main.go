package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastBody struct {
	Type    string `json:"type"`
	Message int    `json:"message"`
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
	Unacked   map[string]map[int]struct{}
	Neighbors []string
}

func NewServerNode() *ServerNode {
	return &ServerNode{
		Node:      maelstrom.NewNode(),
		Messages:  make(map[int]bool),
		Unacked:   make(map[string]map[int]struct{}),
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

		if !seen {
			for _, neighbor := range n.Neighbors {
				n.Unacked[neighbor][body.Message] = struct{}{}
			}
		}
		n.Unlock()

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
		for _, neighbor := range n.Neighbors {
			n.Unacked[neighbor] = make(map[int]struct{})
		}
		n.Unlock()

		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	go func() {
		for {
			time.Sleep(500 * time.Millisecond)

			n.Lock()
			toSend := make(map[string][]int)
			for _, neighbor := range n.Neighbors {
				for msg := range n.Unacked[neighbor] {
					toSend[neighbor] = append(toSend[neighbor], msg)
				}
			}
			n.Unlock()

			for neighbor, msgs := range toSend {
				for _, unackedMsg := range msgs {
					req := BroadcastBody{
						Type:    "broadcast",
						Message: unackedMsg,
					}
					n.RPC(neighbor, req, func(msg maelstrom.Message) error {
						n.Lock()
						defer n.Unlock()

						delete(n.Unacked[neighbor], unackedMsg)
						return nil
					})
				}
			}
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
