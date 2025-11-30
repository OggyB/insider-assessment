package cache

import "fmt"

type Prefix string

const (
	SentMessages Prefix = "sent_messages"
)

func (p Prefix) Key(id string) string {
	return fmt.Sprintf("%s:%s", p, id)
}
