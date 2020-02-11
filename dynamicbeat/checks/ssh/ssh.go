package ssh

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/s-newman/scorestack/dynamicbeat/checks/schema"
	"golang.org/x/crypto/ssh"
)

// The Definition configures the behavior of the SSH check
// it implements the "check" interface
type Definition struct {
	ID           string // unique identifier for this check
	Name         string // a human-readable title for the check
	Group        string // (required) The group ID
	IP           string // (required) IP of the host to run the ICMP check against
	Username     string // (required) The user to login with over ssh
	Password     string // (required) The password for the user that you wish to login with
	Cmd          string // (required) The command to execute once ssh connection established
	MatchContent bool   // (optional, default=false) Whether or not to match content like checking files
	ContentRegex string // (optional, default=`.*`) Regex to match if reading a file
	Port         string // (optional, default=22) The port to attempt an ssh connection on
}

// Run a single instance of the check
func (d *Definition) Run(wg *sync.WaitGroup, out chan<- schema.CheckResult) {
	defer wg.Done()

	// Set up result
	result := schema.CheckResult{
		Timestamp: time.Now(),
		ID:        d.ID,
		Name:      d.Name,
		CheckType: "ssh",
	}

	// Config SSH client
	config := &ssh.ClientConfig{
		User: d.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(d.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// Create the ssh client
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", d.IP, d.Port), config)
	if err != nil {
		result.Message = fmt.Sprintf("Error creating ssh client: %s", err)
		out <- result
		return
	}

	// Create a session from the connection
	session, err := client.NewSession()
	if err != nil {
		result.Message = fmt.Sprintf("Error creating a ssh session: %s", err)
		out <- result
		return
	}
	defer session.Close()

	// Run a command
	output, err := session.CombinedOutput(d.Cmd)
	if err != nil {
		result.Message = fmt.Sprintf("Error executing command: %s", err)
		out <- result
		return
	}

	// Check if we are going to match content
	if !d.MatchContent {
		// If we made it here the check passes
		result.Message = fmt.Sprintf("Command %s executed successfully: %s", d.Cmd, output)
		result.Passed = true
		out <- result
		return
	}

	// Match some content
	regex, err := regexp.Compile(d.ContentRegex)
	if err != nil {
		result.Message = fmt.Sprintf("Error compiling regex string %s : %s", d.ContentRegex, err)
		out <- result
		return
	}

	// Check if the content matches
	if !regex.Match(output) {
		result.Message = fmt.Sprintf("Matching content not found")
		out <- result
		return
	}

	// If we reach here the check is successful
	result.Passed = true
	out <- result
}

// Init the check using a known ID and name. The rest of the check fields will
// be filled in by parsing a JSON string representing the check definition.
func (d *Definition) Init(id string, name string, def []byte) error {

	// Set ID and Name
	d.ID = id
	d.Name = name

	// Unpack JSON definition
	err := json.Unmarshal(def, &d)
	if err != nil {
		return err
	}

	// Check for optional Port value
	if d.Port == "" {
		d.Port = "22"
	}

	// Check for optional ContentRegex
	if d.ContentRegex == "" {
		d.ContentRegex = ".*"
	}

	// Check for missing fields
	missingFields := make([]string, 0)
	if d.IP == "" {
		missingFields = append(missingFields, "IP")
	}

	if d.Username == "" {
		missingFields = append(missingFields, "Username")
	}

	if d.Password == "" {
		missingFields = append(missingFields, "Password")
	}

	if d.Cmd == "" {
		missingFields = append(missingFields, "Cmd")
	}

	// Error only the first missing field, if there are any
	if len(missingFields) > 0 {
		return schema.ValidationError{
			ID:    d.ID,
			Type:  "ssh",
			Field: missingFields[0],
		}
	}
	return nil
}
