package test

import (
	"context"
	"fmt"
	"git.moresec.cn/zhangtian/ktmt"
	"git.moresec.cn/zhangtian/ktmt/client"
	"git.moresec.cn/zhangtian/ktmt/server"
	"net"
	"testing"
	"time"
)

func init() {
	ln, err := net.Listen("tcp", ":2022")
	if err != nil {
		panic(err)
	}
	srv, err := server.NewApp(ktmt.NewJsonPresentationLayer(),
		server.NewOptions(ln,
			server.WithOnOnline(func(s string) error {
				fmt.Println("xxx")
				return nil
			}), server.WithOnOffline(func(s string) error {
				fmt.Println("offline")
				return nil
			}),
		))
	if err != nil {
		panic(err)
	}
	srv.SetDefaultHandler(func(message ktmt.Message) bool {
		fmt.Println(message.Payload())
		return true
	})
	err = srv.Open(context.Background())
	if err != nil {
		panic(err)
	}
}

func BenchmarkConnect(b *testing.B) {
	b.SetParallelism(50)
	b.RunParallel(func(pb *testing.PB) {
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		cli, err := client.NewApp(ktmt.NewJsonPresentationLayer(),

			client.NewOptions(id, func() (conn net.Conn, err error) {
				return net.Dial("tcp", "127.0.0.1:2022")
			}))
		if err != nil {
			//b.Error(err)
			//return
			fmt.Println(err)
		}
		defer cli.Close()
		err = cli.Open(context.Background())
		if err != nil {

			b.Error(err)
			panic(err)
		}

		for pb.Next() {
			err = cli.PublishTimeout(context.TODO(), "", "sdadsadasd", time.Second)
			if err != nil {
				b.Error(err)
				panic(err)
			}
		}
	})
}
