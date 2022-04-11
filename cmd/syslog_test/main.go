package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/crewjam/rfc5424"
)

func main() {
	network := flag.String("n", "tcp", "tcp or udp")
	raddr := flag.String("r", "", "localhost:1234")
	host := flag.String("h", "", "host")

	flag.Parse()

	if *host == "" {
		var err error
		*host, err = os.Hostname()
		if err != nil {
			panic(err)
		}
	}

	conn, err := net.Dial(*network, *raddr)
	if err != nil {
		panic(err)
	}

	var str string
	for {
		_, err := fmt.Scan(&str)
		if err != nil {
			panic(err)
		}
		if str == ".exit" {
			return
		}

		m := rfc5424.Message{
			Priority:  rfc5424.Daemon | rfc5424.Info,
			Timestamp: time.Now(),
			Hostname:  *host,
			AppName:   "test",
			Message:   []byte(str),
		}

		data, err := m.MarshalBinary()
		if err != nil {
			panic(err)
		}
		conn.Write(data)
	}

}
