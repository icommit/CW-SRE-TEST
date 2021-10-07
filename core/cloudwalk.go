/*
Package core is the workhorse for this project. The package define internal and
exportable functions for our main program. The functions to create a firestore client
send email notification message, make connection with tcp and http echo servers and return their state
and a wrapper function for the return http and tcp state an their logs is defined here.
*/
package core

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/icommit/SRETest/pkg/models"
	"github.com/mailgun/mailgun-go/v4"
	"gopkg.in/yaml.v3"
)

var msg_tcp string  //failure/success message tcp
var msg_http string //failure/success message http

// Helper function properly get items from path.
func Currentdir() (cwd string) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	return cwd
}

// Helper function to read our yaml file and parse its fields to Config Struct
func ReadConf(f string) (*models.Config, error) {
	buf, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	c := &models.Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, fmt.Errorf("in file %q: %v", f, err)
	}

	return c, nil
}

// Helper function to send email using Mailgun.
func sendMail(ctx context.Context, domain string, privateKey string, sender string, recipient string, subject string, body string) {
	mg := mailgun.NewMailgun(domain, privateKey)
	message := mg.NewMessage(sender, subject, body, recipient)
	rtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	resp, id, err := mg.Send(rtx, message)
	if err != nil {
		log.Fatal(err)
	}
	m_msg := fmt.Sprintf("%s: ID: %s Resp: %s\n", time.Now(), id, resp)
	log.Printf(m_msg)
}

// Creates a firestore client.
func CreateClient(ctx context.Context) *firestore.Client {
	projectID := "cloudwalk-sre-test"
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	return client
}

// Http Endpoint: Establishes connection to Http-Echo server.
// Returns a boolean for whether the proper response was received as well as
// Http specifc logs, and status data stored in firebase.
// Since spaces in the url will cause a panic, the message sent on this endpoint is trimmed to remove spaces.
func HttpState(url string, auth string, msg string, timeOut int) (bool, models.GLogs, models.Status) {
	ctx := context.Background()
	crt := CreateClient(ctx)
	defer crt.Close()
	store := crt.Collection("current_status").Doc("http")
	note := crt.Collection("config").Doc("config")
	trim := strings.ReplaceAll(msg, " ", "") //we don't want spaces in our http url
	res_msg := fmt.Sprintf("CLOUDWALK %s", trim)
	link := fmt.Sprintf("%s/?auth=%s&buf=%s", url, auth, trim)
	var http_logs models.GLogs
	var http_stat models.Status
	var notify models.Notification

	t := time.Now().Format("Mon Jan _2 15:04:05 2006")

	dc, err := store.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to Get document: %v", err)
	}
	dc.DataTo(&http_stat) // read from cloudfirestore into GLogs

	nt, err := note.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to Get document: %v", err)
	}
	nt.DataTo(&notify) // Reads from firestore into Notification collection
	http_logs.Email = notify.Email
	http_logs.Update = notify.Update

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Println("Request Error ", err)
	} else {
		auth_ok := fmt.Sprintf("%s: %s", t, "Auth Token Accepted")
		sent_msg := fmt.Sprintf("%s: Sent: %s", t, trim)
		http_logs.Auth = auth_ok
		http_logs.Sent = sent_msg
	}
	client := http.Client{
		Timeout: time.Duration(timeOut) * time.Second, //10 seconds
	}
	res, err := client.Do(req)
	if err != nil {
		log.Println("Error on response \n[Error]: ", err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Error reading bytes: ", err)
	}
	fmtBody := strings.Replace(string(body), "\n", "", -1)
	m := strings.Replace(fmtBody, "\t", "", -1) //final sanitize
	is_up := m == res_msg
	log.Println("http: ", is_up)
	rec_msg := fmt.Sprintf("%s: Received: %s", t, m)
	http_logs.Received = rec_msg
	up_down := fmt.Sprintf("%s: Connection Active: %t", t, is_up)
	http_logs.State = up_down
	http_logs.CloudState = is_up

	http_logs.Threshold = msg_http
	return is_up, http_logs, http_stat
}

// Tcp Endpoint: TcpState Establishes connection with Tcp-Echo server and return the state, logs and
// general status of the server stored in firestore.
func TcpState(host string, port string, auth string, msg string, timeOut int) (bool, models.GLogs, models.Status) {
	ctx := context.Background()
	crt := CreateClient(ctx)
	defer crt.Close()
	store := crt.Collection("current_status").Doc("tcp")
	var tcp_stat models.Status
	var tcp_logs models.GLogs

	t := time.Now().Format("Mon Jan _2 15:04:05 2006")

	dc, err := store.Get(ctx)
	if err != nil {
		log.Fatalf("Failed to Get document: %v", err)
	}
	dc.DataTo(&tcp_stat)

	res_msg := append([]byte("CLOUDWALK "), []byte(msg)...)
	out := net.Dialer{
		Timeout: time.Duration(timeOut) * time.Second,
	}
	conn, err := out.Dial("tcp", host+":"+port)
	defer conn.Close()
	if err != nil {
		fmt.Println("Error Connecting: ", err.Error())
		os.Exit(1)
	}
	for {
		text := fmt.Sprintf("auth %s", auth)
		//send := append([]byte("auth "), []byte(pass)...)
		fmt.Fprintf(conn, text+"\n")
		message, _ := bufio.NewReader(conn).ReadString('\n')
		if message == "auth ok"+"\n" {
			auth_ok := fmt.Sprintf("%s: %s", t, "Auth Token Accepted")
			tcp_logs.Auth = auth_ok

			fmt.Println("Auth Ok")
			fmt.Fprintf(conn, msg+"\n")
			log.Printf("Send: %s", msg)
			sent_msg := fmt.Sprintf("%s: Sent: %s", t, msg)
			tcp_logs.Sent = sent_msg

			m, _ := bufio.NewReader(conn).ReadBytes('\n')
			log.Printf("Receive: %s", m)
			rec_msg := fmt.Sprintf("%s: Received: %s", t, m)
			tcp_logs.Received = rec_msg

			fmtBody := strings.Replace(string(m), "\n", "", -1)
			san := strings.Replace(fmtBody, "\t", "", -1) //final sanitize
			is_up := san == string(res_msg)
			log.Println("tcp: ", is_up)
			up_down := fmt.Sprintf("%s: Connection Active: %t", t, is_up)
			tcp_logs.State = up_down
			tcp_logs.Threshold = msg_tcp
			tcp_logs.CloudState = is_up
			return is_up, tcp_logs, tcp_stat
		} else {
			auth_ok := fmt.Sprintf("%s: %s", t, "Wrong Auth Token")
			tcp_logs.Auth = auth_ok
			return false, tcp_logs, tcp_stat
		}
	}
}

// Checks returns a helper function that represents either HttpState or TcpState. Alongside the function returned
// is our log warehouse that combines http and tcp logs and data. In here, the logic to increment
// and decrement health thresholds as well as to send email notification if the thresholds are reached is defined.
// the tester must explicitly opt in to recieve email notification by entering an email and consenting to receive
// email notification in the frontend.
func Checks(ctx context.Context, client *firestore.Client, service_type string, interval int,
	healthy_threshold int, unhealthy_threshold int) (func(bool, models.GLogs, models.Status), models.LogWarehouse) {
	store := client.Collection("current_status").Doc(service_type)
	note := client.Collection("config").Doc("config")
	var check_logs models.LogWarehouse
	var notify models.Notification
	nested := func(f bool, i models.GLogs, h models.Status) {

		t := time.Now().Format("Mon Jan _2 15:04:05 2006")
		msg_http = ""
		msg_tcp = ""
		var status models.Status
		dc, err := store.Get(ctx)
		if err != nil {
			log.Fatalf("Failed to Get document: %v", err)
		}
		dc.DataTo(&status)

		nt, err := note.Get(ctx)
		if err != nil {
			log.Fatalf("Failed to Get document: %v", err)
		}
		nt.DataTo(&notify)

		is_up := f

		// what to do when server state is unhealthy while is still down; Up;
		if status.State == "unhealthy" {
			if !is_up {
				_, err := store.Set(ctx, map[string]interface{}{
					"uptime_count":   0,
					"downtime_count": 0,
				}, firestore.MergeAll)
				if err != nil {
					log.Printf("Set: An error has occurred: %s", err)
				}
			} else {
				_, err := store.Update(ctx, []firestore.Update{{
					Path: "uptime_count", Value: firestore.Increment(1),
				}})
				if err != nil {
					log.Printf("Update: An error has occurred: %s", err)
				}
			}
		}

		// what to do when server is healthy but down; or up.
		if status.State == "healthy" {
			if !is_up {
				_, err := store.Update(ctx, []firestore.Update{{
					Path: "downtime_count", Value: firestore.Increment(1),
				}})
				if err != nil {
					log.Printf("Update: An error has occurred: %s", err)
				}
			} else {
				_, err := store.Set(ctx, map[string]interface{}{
					"downtime_count": 0,
					"uptime_count":   0,
				}, firestore.MergeAll)
				if err != nil {
					log.Printf("Set: An error has occurred: %s", err)
				}

			}
		}
		// called when unhealthy threshold is reached
		if status.Downtime == unhealthy_threshold {
			_, err := store.Set(ctx, map[string]interface{}{
				"state": "unhealthy",
			}, firestore.MergeAll)
			if err != nil {
				log.Printf("Set: failed to update healthy state: %s", err)
			}
			_, er := store.Set(ctx, map[string]interface{}{
				"timestamp": firestore.ServerTimestamp,
			}, firestore.MergeAll)
			if er != nil {
				log.Printf("Failed to update timestamp: %s", err)
			}

			_, e := store.Set(ctx, map[string]interface{}{
				"downtime_count": 0,
				"uptime_count":   0,
			}, firestore.MergeAll)
			if e != nil {
				log.Printf("Set: An error has occurred: %s", err)
			}
			if notify.Update && notify.Email != "" {
				thresh_msg := fmt.Sprintf("%s: Failure Threshold Reached. %s Server is Down. Confirmation Sent!", t, service_type)
				msg_tcp = thresh_msg
				msg_http = thresh_msg

				subject := service_type + " Echo Server Down!"
				body := service_type + " Echo server down. Maximum failure threshold reached" + "\n" + "Will try to make contact again....."

				C, err := ReadConf(filepath.Base("../app.yaml"))
				if err != nil {
					log.Fatal(err)
				}
				sendMail(ctx, C.Handlers.Domain, C.Handlers.APIKey, C.Handlers.Sender, notify.Email, subject, body)
			} else {
				thresh_msg2 := fmt.Sprintf("%s: Failure Threshold Reached. %s Server is Down.", t, service_type)
				msg_tcp = thresh_msg2
				msg_http = thresh_msg2
			}
		}
		// called when healthy threshold reached
		if status.Uptime == healthy_threshold {
			_, err := store.Set(ctx, map[string]interface{}{
				"state": "healthy",
			}, firestore.MergeAll)
			if err != nil {
				log.Printf("Set: failed to update healthy state: %s", err)
			}
			_, er := store.Set(ctx, map[string]interface{}{
				"timestamp": firestore.ServerTimestamp,
			}, firestore.MergeAll)
			if er != nil {
				log.Printf("Failed to update timestamp: %s", err)
			}

			_, e := store.Set(ctx, map[string]interface{}{
				"uptime_count":   0,
				"downtime_count": 0,
			}, firestore.MergeAll)
			if e != nil {
				log.Printf("Set: An error has occurred: %s", err)
			}
			if notify.Update && notify.Email != "" {
				thresh_msg := fmt.Sprintf("%s: Success Threshold Reached! %s Server is Up. Confirmation Sent!", t, service_type)

				subject := service_type + " Echo Server Back Online!"
				body := service_type + " Echo server Back up. Maximum success threshold reached" + "\n" + "Scanning....."

				C, err := ReadConf(filepath.Base("../app.yaml"))
				if err != nil {
					log.Fatal(err)
				}
				sendMail(ctx, C.Handlers.Domain, C.Handlers.APIKey, C.Handlers.Sender, notify.Email, subject, body)
				msg_tcp = thresh_msg
				msg_http = thresh_msg
			} else {
				thresh_msg2 := fmt.Sprintf("%s: Success Threshold Reached! %s Server is Up!", t, service_type)
				msg_tcp = thresh_msg2
				msg_http = thresh_msg2
			}
		}

	}
	return nested, check_logs
}
