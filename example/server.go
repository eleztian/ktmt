package main

import (
	"context"
	"fmt"
	"git.moresec.cn/zhangtian/ktmt"
	"git.moresec.cn/zhangtian/ktmt/server"
	"net"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:5001")
	if err != nil {
		panic(err)
	}

	app, err := server.NewApp(
		ktmt.NewJsonPresentationLayer(),
		server.NewOptions(ln,
			server.WithOnOnline(func(s string) error {
				fmt.Printf("online %s %v \n", s, time.Now())
				return nil
			}),
			server.WithOnOffline(func(s string) error {
				fmt.Printf("offline %s %v \n", s, time.Now())
				return nil
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
		err = app.Publish(ctx, "/test", []string{"world"})
		if err != nil {
			panic(err)
		}
		return true
	})

	err = app.Open(ctx)

	time.Sleep(time.Second * 1)
	select {}
	for {
		err = app.Publish(ctx, "/test", "world")
		if err != nil {
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}

}
