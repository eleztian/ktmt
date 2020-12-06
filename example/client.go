package main

import (
	"context"
	"fmt"
	"github.com/eleztian/ktmt"
	"github.com/eleztian/ktmt/client"
	"net"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:5001")
	}

	app, err := client.NewApp(
		ktmt.NewJsonPresentationLayer(),
		client.NewOptions(
			"test-id", dialer,
			client.WithKeepalive(1),
			client.WithOnClose(func(id string) {
				fmt.Println("closed")
			}),
		),
	)
	if err != nil {
		panic(err)
	}
	defer app.Close()

	app.AddRoute("/test", func(message ktmt.Message) bool {
		fmt.Println(string(message.Payload()))
		time.Sleep(time.Second)

		err = app.Publish(ctx, "/test", []string{"hello"})
		if err != nil {
			panic(err)
		}
		return true
	})

	err = app.Open(ctx)
	if err != nil {
		panic(err)
	}

	err = app.Publish(ctx, "/test", []string{"hello"})
	if err != nil {
		panic(err)
	}
	select {}
	for {
		err = app.Publish(ctx, "/test", []string{"hello"})
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Second)
	}
}
