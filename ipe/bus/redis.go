package bus

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	log "github.com/golang/glog"
	"github.com/redis/go-redis/v9"
)

type rbus struct {
	sync.Mutex

	ctx     context.Context
	redis   *redis.Client
	channel string
	chs     map[string]chan Message
}

func NewRedisBus(
	ctx context.Context,
	dsn string,
	channel string,
) (Bus, error) {
	var b Bus
	o, err := redis.ParseURL(dsn)
	if err != nil {
		return b, err
	}
	rds := redis.NewClient(o)

	return &rbus{
		ctx:   ctx,
		redis: rds,
		chs:   map[string]chan Message{},
	}, nil
}

func (b *rbus) Close() {
	b.redis.Close()
	for _, ch := range b.chs {
		close(ch)
	}
}

func (b *rbus) Publish(m Message) error {
	return b.redis.Publish(b.ctx, b.channel, m).Err()
}

func (b *rbus) Channel(appID string) (chan Message, error) {
	b.Lock()
	defer b.Unlock()

	ch, exists := b.chs[appID]
	if exists {
		return ch, nil
	}

	ch = make(chan Message)
	b.chs[appID] = ch

	pb := b.redis.PSubscribe(b.ctx, b.channel)

	go func() {
		defer pb.Close()

		for rm := range pb.Channel() {
			m := Message{}
			if err := json.NewDecoder(strings.NewReader(rm.Payload)).Decode(&m); err != nil {
				log.Errorf("Can't decode %s into Message", rm.Payload)
				continue
			}
			ch <- m
		}
	}()

	return b.chs[appID], nil
}
