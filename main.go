package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/icommit/SRETest/core"
	"github.com/icommit/SRETest/pkg/models"
)

var warehouse models.LogWarehouse // Combined logs for http and tcp
var p []models.GLogs              // Logs for http endpoint
var q []models.GLogs              // Logs for tcp endpoint

func handleRequest() {
	http.HandleFunc("/", home)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Concurrently run HttpState function from core and pause for interval "t"
// Assign generated logs for current run to the appropriate logwarehouse entry for http.
func concurrent_http(f func(bool, models.GLogs, models.Status), t time.Duration) {
	for {
		time.Sleep(t * time.Second)
		C, err := core.ReadConf("./app.yaml")
		if err != nil {
			log.Fatal(err)
		}

		// We want the first 500 log items before reseting our collection.
		if len(p) == 500 {
			p = nil
		}

		token := C.Handlers.Token
		timeout := C.Handlers.Timeout
		msg := C.Handlers.Msg
		http, i, h := core.HttpState(C.Handlers.HttpUrl, token, msg, timeout)
		warehouse.ClientLogs = i
		warehouse.StatusLogs = h
		warehouse.Notification.Email = i.Email
		warehouse.Notification.Update = i.Update

		p = append(p, i)
		warehouse.LogSlice = p

		go f(http, i, h) // run function in its own goroutine
	}
}

// Concurrently run TcpState from core and pause for interval amount "t".
// Assign generated logs for the current run to the appropriate log warehouse for tcp.
func concurrent_tcp(f func(bool, models.GLogs, models.Status), t time.Duration) {
	for {
		time.Sleep(t * time.Second)

		C, err := core.ReadConf("./app.yaml")
		if err != nil {
			log.Fatal(err)
		}
		// we want the first 500 log items to display in our frontend before freeing memory.
		if len(q) == 500 {
			q = nil
		}

		host := C.Handlers.TcpUrl
		port := C.Handlers.Port
		token := C.Handlers.Token
		timeout := C.Handlers.Timeout
		msg := C.Handlers.Msg
		tcp, i, w := core.TcpState(host, port, token, msg, timeout)
		warehouse.TcpLogWarehouse.ClientLogs = i
		warehouse.TcpLogWarehouse.StatusLogs = w

		q = append(q, i)
		warehouse.TcpLogWarehouse.LogSlice = q
		go f(tcp, i, w)
	}
}

func main() {
	ctx := context.Background()
	C, err := core.ReadConf("./app.yaml")
	if err != nil {
		log.Fatal(err)
	}

	i := C.Handlers.Interval
	hThreshold := C.Handlers.HThreshold
	uhThreshold := C.Handlers.UhThreshold

	client := core.CreateClient(ctx)
	defer client.Close()

	// core Check function for tcp
	a, t := core.Checks(ctx, client, "tcp", i, hThreshold, uhThreshold)

	// core check function for http
	b, h := core.Checks(ctx, client, "http", i, hThreshold, uhThreshold)
	warehouse = h
	warehouse.TcpLogWarehouse = t.TcpLogWarehouse

	go concurrent_tcp(a, time.Duration(i))  // pass in tcp client and run in a separate thread
	go concurrent_http(b, time.Duration(i)) // pass in http and run in a separate thread
	handleRequest()                         // the fun begins
	time.Sleep(1 * time.Second)
}
