package models

import "time"

// Global Configurator. This collection contains fields that are specified for our program.
// The struct contains fields parsed from our app.yaml file.
type Config struct {
	Handlers struct {
		Token       string `yaml:"auth_token"`          // authentication token
		TcpUrl      string `yaml:"tcp_url"`             // tcp url
		Port        string `yaml:"port"`                // tcp port
		HttpUrl     string `yaml:"http_url"`            // http url
		Msg         string `yaml:"message"`             // the message to send to the echo servers
		Timeout     int    `yaml:"timeout"`             // timeout for our servers
		Interval    int    `yaml:"interval"`            // how long to pause before resuming execution
		HThreshold  int    `yaml:"healthy_threshold"`   // healthy threshold
		UhThreshold int    `yaml:"unhealthy_threshold"` // unhealthy threshold

		Sender    string `yaml:"sender"`    // Email Notification: Sender email
		Recipient string `yaml:"recipient"` // Recipient. This field is no longer used. Notification collection field is used.
		Domain    string `yaml:"domain"`    // mailgun specific configuration.
		APIKey    string `yaml:"api_key"`   // mailgun api key
	} `yaml:"env_variables"`
}

// Status typed collection holds the data from Firebase Cloud Firestore.
// The struct reads and write data to the "current_status" collection.
type Status struct {
	State     string    `firestore:"state,omitempty"`          // Field is either healthy or unhealthy
	Uptime    int       `firestore:"uptime_count,omitempty"`   // Healthy Threshold field
	Downtime  int       `firestore:"downtime_count,omitempty"` // Unhealthy Threshold field
	Timestamp time.Time `firestore:"timestamp,omitempty"`      // Time at which State Field is updated
}

// Notification reads data from Cloud Firestore "config" collection
// Basically we want to know where to send email notification and whether
// we should stop/start getting email notification. This is on for the purpose fo this demo
// since in real life we want to always get notification for up/downtime of a server
type Notification struct {
	Email  string `firestore:"email,omitempty"`  // Email to receive server status messages
	Update bool   `firestore:"update,omitempty"` // Stop/Start receiving notification
}

// General Logs struct is a collection of log output to display in the web frontend
// Each server holds a collection of GLogs to keep tract of important metrics.
// For the purpose of this demonstration, Logs are persisted in memory. In real life we
// will persist logs in nonvolatile memory like database or hardrive.
type GLogs struct {
	Auth       string // Log for auth token success of failure
	Sent       string // Log for the message sent to the server
	Received   string // Log message echoed from server
	State      string // Log to indicate server status on each run. Accumulates for Threshold field.
	Threshold  string // Log for success/failure threshold
	CloudState bool   // Single bool. used in html template to know how to properly display element.
	Email      string // Fed to Notification Struct Email Field. Only available for http server
	Update     bool   // Fed to Nofification Update Field.
}

// A collection of all our logs and data to display in web frontend for the Tcp Echo Server
type TcpLogWarehouse struct {
	ClientLogs GLogs
	StatusLogs Status
	LogSlice   []GLogs
}

// Global Log Warehouse that. Contains all logs and data for both tcp and http.
// This struct is passed to our template.
type LogWarehouse struct {
	Notification    Notification
	ClientLogs      GLogs
	StatusLogs      Status
	LogSlice        []GLogs
	TcpLogWarehouse TcpLogWarehouse
}
