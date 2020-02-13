package ldap

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/s-newman/scorestack/dynamicbeat/checks/schema"
	"gopkg.in/ldap.v2"
)

// The Definition configures the behavior of the SSH check
// it implements the "check" interface
type Definition struct {
	ID       string // a unique identifier for this check
	Name     string // a human-readable title for the check
	Group    string // the group this check is part of
	User     string // (required) The user written in DN syntax
	Password string // (required) the password for the user
	Fqdn     string // (required) The Fqdn of the ldap server
	Ldaps    bool   // (optional, default=false) Whether or not to use LDAP+TLS
	Port     string // (optional, default=389) Port for ldap
}

// Run a single instance of the check
func (d *Definition) Run(wg *sync.WaitGroup, out chan<- schema.CheckResult) {
	defer wg.Done()

	// Set up result
	result := schema.CheckResult{
		Timestamp: time.Now(),
		ID:        d.ID,
		Group:     d.Group,
		CheckType: "ldap",
	}

	// Set timeout
	ldap.DefaultTimeout = 5 * time.Second

	// Normal, default ldap check
	lconn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", d.Fqdn, d.Port))
	if err != nil {
		result.Message = fmt.Sprintf("Could not dial server %s : %s", d.Fqdn, err)
		out <- result
		return
	}
	defer lconn.Close()

	// Set message timeout
	lconn.SetTimeout(5 * time.Second)

	// Add TLS if needed
	if d.Ldaps {
		err = lconn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			result.Message = fmt.Sprintf("TLS session creation failed : %s", err)
			out <- result
			return
		}
	}

	// Attempt to login
	err = lconn.Bind(d.User, d.Password)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to login with user %s : %s", d.User, err)
		out <- result
		return
	}

	// If we reached here the check passes
	result.Passed = true
	out <- result
}

// Init the check using a known ID and name. The rest of the check fields will
// be filled in by parsing a JSON string representing the check definition.
func (d *Definition) Init(id string, name string, group string, def []byte) error {

	// Explicitly set default values
	d.Port = "389"

	// Unpack JSON definition
	err := json.Unmarshal(def, &d)
	if err != nil {
		return err
	}

	// Set generic values
	d.ID = id
	d.Name = name
	d.Group = group

	// Check for missing fields
	missingFields := make([]string, 0)
	if d.User == "" {
		missingFields = append(missingFields, "User")
	}

	if d.Password == "" {
		missingFields = append(missingFields, "Password")
	}

	if d.Fqdn == "" {
		missingFields = append(missingFields, "Fqdn")
	}

	if len(missingFields) > 0 {
		return schema.ValidationError{
			ID:    d.ID,
			Type:  "ldap",
			Field: missingFields[0],
		}
	}
	return nil
}