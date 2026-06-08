package main

import (
	"maps"
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type SendRequestBody struct {
	maelstrom.MessageBody
	Key string `json:"key"`
	Msg int    `json:"msg"`
}

type SendResponseBody struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
}

type PollRequestBody struct {
	maelstrom.MessageBody
	Offsets map[string]int `json:"offsets"`
}

type PollResponseBody struct {
	Type string             `json:"type"`
	Msgs map[string][][2]int `json:"msgs"`
}

type CommitOffsetsRequestBody struct {
	maelstrom.MessageBody
	Offsets map[string]int `json:"offsets"`
}

type ListCommittedOffsetsRequestBody struct {
	maelstrom.MessageBody
	Keys []string `json:"keys"`
}

type ListCommittedOffsetsResponseBody struct {
	Type    string         `json:"type"`
	Offsets map[string]int `json:"offsets"`
}

type ServerNode struct {
	*maelstrom.Node
	sync.Mutex
	CommittedOffsets map[string]int
	Logs             map[string][]int
}

func NewServerNode() *ServerNode {
	node := maelstrom.NewNode()

	return &ServerNode{
		Node: node,
		CommittedOffsets: make(map[string]int),
		Logs: make(map[string][]int),
	}
}

func main() {
	n := NewServerNode()

	n.Handle("send", func(msg maelstrom.Message) error {
		var body SendRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		n.Logs[body.Key] = append(n.Logs[body.Key], body.Msg)
		offset := len(n.Logs[body.Key]) - 1
		n.Unlock()

		resp := SendResponseBody{
			Type: "send_ok",
			Offset: offset,
		}

		return n.Reply(msg, resp)		
	})

	n.Handle("poll", func(msg maelstrom.Message) error {
		var body PollRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		msgs := make(map[string][][2]int)
		for key, offset := range body.Offsets {
			for i := offset; i < len(n.Logs[key]); i++ {
				msgs[key] = append(msgs[key], [...]int{i, n.Logs[key][i]})
			}
		}
		n.Unlock()

		resp := PollResponseBody{
			Type: "poll_ok",
			Msgs: msgs,
		}
	
		return n.Reply(msg, resp)
	})

	n.Handle("commit_offsets", func(msg maelstrom.Message) error {
		var body CommitOffsetsRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		maps.Copy(n.CommittedOffsets, body.Offsets)
		n.Unlock()

		return n.Reply(msg, maelstrom.MessageBody{Type: "commit_offsets_ok"})
	})

	n.Handle("list_committed_offsets", func(msg maelstrom.Message) error {
		var body ListCommittedOffsetsRequestBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		n.Lock()
		offsets := make(map[string]int)
		for _, key := range body.Keys {
			if _, ok := n.CommittedOffsets[key]; ok {
				offsets[key] = n.CommittedOffsets[key]
			}
		}
		n.Unlock()

		resp := ListCommittedOffsetsResponseBody{
			Type: "list_committed_offsets_ok",
			Offsets: offsets,
		}
		
		return n.Reply(msg, resp)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
