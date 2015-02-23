package main

import (
	"fmt"
	"github.com/cloudfoundry/cli/cf/errors"
	"github.com/cloudfoundry/cli/generic"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/cf/terminal"
	"io/ioutil"
	"os"
	"net/http"
	"strings"
	"net/url"
)

type BrooklynPlugin struct{
	ui            terminal.UI
	cliConnection plugin.CliConnection
	yamlMap       generic.Map
}

type BrokerCredentials struct{
	broker    string 
	username  string
	password  string
}

func NewBrokerCredentials(broker, username, password string) *BrokerCredentials{
	return &BrokerCredentials{broker, username, password}
}

func SendRequest(req *http.Request) ([]byte, error){
	client := &http.Client{}
    resp, err := client.Do(req)
    AssertErrorIsNil(err)
    defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if resp.Status != "200 OK" {
    	fmt.Println("response Status:", resp.Status)
    	fmt.Println("response Headers:", resp.Header)
    	fmt.Println("response Body:", string(body))
	}
	return body, err
}

func ServiceBrokerUrl(cliConnection plugin.CliConnection, broker string) (string, error){
	brokers, err := cliConnection.CliCommandWithoutTerminalOutput("service-brokers")
	Assert(err == nil, "")
	for _, a := range brokers {
		fields := strings.Fields(a)	
		if fields[0] == broker { 
			return fields[1], nil
		}
	}
	return "", errors.New("No such broker")
}

func CreateRestCallUrlString(cliConnection plugin.CliConnection, cred *BrokerCredentials, path string) string{
	brokerUrl, err := ServiceBrokerUrl(cliConnection, cred.broker)
	Assert(err == nil, "No such broker")
	brooklynUrl, err := url.Parse(brokerUrl)
	Assert(err == nil, "")	
	brooklynUrl.Path = path
	brooklynUrl.User = url.UserPassword(cred.username, cred.password)
	return brooklynUrl.String()
}


func Assert(cond bool, message string) {
	if !cond {
		panic(errors.New("PLUGIN ERROR: " + message))
	}
}

func AssertErrorIsNil(err error) {
	if err != nil {
		Assert(false, "error not nil, "+err.Error())
	}
}

func (c *BrooklynPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println(r)
        }
    }()
	c.ui = terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	c.cliConnection = cliConnection
	switch args[1] {
	case "push":
		NewPushCommand(cliConnection).push(args[1:])
		//c.push(args[1:])
	case "add-catalog":
		Assert(len(args) == 6, "incorrect number of arguments")
		NewAddCatalogCommand(cliConnection).addCatalog(NewBrokerCredentials(args[2], args[3], args[4]), args[5])
		defer fmt.Println("Catalog item sucessfully added.")
	case "delete-catalog":
		Assert(len(args) == 7, "incorrect number of arguments")
		NewAddCatalogCommand(cliConnection).deleteCatalog(NewBrokerCredentials(args[2], args[3], args[4]), args[5], args[6])
	case "effectors":
		Assert(len(args) == 6, "incorrect number of arguments")
		NewEffectorCommand(cliConnection).listEffectors(NewBrokerCredentials(args[2], args[3], args[4]), args[5])
	case "invoke":
		Assert(len(args) >= 7, "incorrect number of arguments")
		NewEffectorCommand(cliConnection).invokeEffector(NewBrokerCredentials(args[2], args[3], args[4]), args[5], args[6], args[7:])
	case "sensors":
		Assert(len(args) == 6, "incorrect number of arguments")
		NewSensorCommand(cliConnection).listSensors(NewBrokerCredentials(args[2], args[3], args[4]), args[5])
	case "ready":
	    Assert(len(args) == 6, "incorrect number of arguments")
	    fmt.Println("Ready:", NewSensorCommand(cliConnection).isServiceReady(NewBrokerCredentials(args[2], args[3], args[4]), args[5]))
	}
	fmt.Println(terminal.ColorizeBold("OK", 32))
	
}

func (c *BrooklynPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "BrooklynPlugin",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     "brooklyn",
				HelpText: "Brooklyn plugin command's help text",
				// UsageDetails is optional
				// It is used to show help of usage of each command
				UsageDetails: plugin.Usage{
					Usage: "brooklyn\n   cf brooklyn",
				},
			},
		},
	}
}

func main() {
	plugin.Start(new(BrooklynPlugin))
}
