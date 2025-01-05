package main

import (
	"flag"

	"github.com/pusher/pusher-http-go"
)

func main() {
	flag.Parse()

	client := pusher.Client{
		AppID:  "1",
		Key:    "278d525bdf162c739803",
		Secret: "7ad3753142a6693b25b9",
		Host:   "localhost:3080",
	}

	err := client.Trigger(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	if err != nil {
		panic(err)
	}
}
