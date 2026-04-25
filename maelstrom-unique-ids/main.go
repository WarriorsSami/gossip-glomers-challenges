package main

import (
	"encoding/json"
	"fmt"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type GenerateBody struct {
	maelstrom.MessageBody
	Id string `json:"id"`
}

func main() {
	n := maelstrom.NewNode()
	var counter int

	n.Handle("generate", func(msg maelstrom.Message) error {
		var body GenerateBody
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body.Type = "generate_ok"
		body.Id = fmt.Sprintf("%s-%d", n.ID(), counter)
		counter++

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
