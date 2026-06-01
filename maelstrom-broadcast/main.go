package main

import (
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastBody struct {
	Type    string `json:"type"`
	Message int    `json:"message"`
}

type BatchBroadcastBody struct {
	Type     string `json:"type"`
	Messages []int  `json:"messages"`
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
				if neighbor != msg.Src {
					n.Unacked[neighbor][body.Message] = struct{}{}
				}
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

		allNodes := make([]string, 0)
		for neighbor := range body.Topology {
			allNodes = append(allNodes, neighbor)
		}

		sort.Strings(allNodes)
		myIdx := 0
		for idx, neighbor := range allNodes {
			if neighbor == n.ID() { 
				myIdx = idx
				break
			}
		}

		n.Lock()
		n.Neighbors = make([]string, 0)
		total := len(allNodes)
		for offset := 1; offset < total; offset *= 2 {
			neighbor := allNodes[(myIdx+offset)%total]
			if neighbor != n.ID() {
				n.Neighbors = append(n.Neighbors, neighbor)
			}
		}

		for _, neighbor := range n.Neighbors {
			n.Unacked[neighbor] = make(map[int]struct{})
		}
		n.Unlock()

		return n.Reply(msg, maelstrom.MessageBody{Type: "topology_ok"})
	})

	n.Handle("batch_broadcast", func(msg maelstrom.Message) error {
		var body BatchBroadcastBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		for _, incomingMsg := range body.Messages {
			if !n.Messages[incomingMsg] {
				n.Messages[incomingMsg] = true
				for _, neighbor := range n.Neighbors {
					if neighbor != msg.Src {
						n.Unacked[neighbor][incomingMsg] = struct{}{}
					} 
				}
			}
		}
		n.Unlock()

		return n.Reply(msg, maelstrom.MessageBody{Type: "batch_broadcast_ok"})
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

			for neighbor, unackedMsgs := range toSend {
				go func (neighbor string, msgs []int) {
					req := BatchBroadcastBody{
						Type:     "batch_broadcast",
						Messages: msgs,
					}
					n.RPC(neighbor, req, func(msg maelstrom.Message) error {
						n.Lock()
						defer n.Unlock()

						for _, unackedMsg := range msgs {
							delete(n.Unacked[neighbor], unackedMsg)
						}
						return nil
					})
				}(neighbor, unackedMsgs)
			}
		}
	}()

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
