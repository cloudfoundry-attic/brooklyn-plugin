package sensors

import (
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/cf/terminal"
	"net/http"
	"encoding/json"
	"strconv"
)

type SensorCommand struct {
	cliConnection plugin.CliConnection
	ui            terminal.UI
}

func NewSensorCommand(cliConnection plugin.CliConnection, ui terminal.UI) *SensorCommand{
	command := new(SensorCommand)
	command.cliConnection = cliConnection
	return command
}

func (c *SensorCommand) getSensors(cred *broker.BrokerCredentials, service string) map[string]interface{} {
	guid, err := c.cliConnection.CliCommandWithoutTerminalOutput("service", service, "--guid")
	url := broker.CreateRestCallUrlString(c.cliConnection, cred, "sensors/" + guid[0])
	req, err := http.NewRequest("GET", url, nil)
	assert.ErrorIsNil(err)
	body, _ := broker.SendRequest(req)
	//fmt.Println(string(body))
	var sensors map[string]interface{}
	err = json.Unmarshal(body, &sensors)
	assert.ErrorIsNil(err)
	return sensors
}

func (c *SensorCommand) IsServiceReady(cred *broker.BrokerCredentials, service string) bool {
	guid, err := c.cliConnection.CliCommandWithoutTerminalOutput("service", service, "--guid")
	url := broker.CreateRestCallUrlString(c.cliConnection, cred, "is-running/" + guid[0])
	req, err := http.NewRequest("GET", url, nil)
	assert.ErrorIsNil(err)
	body, _ := broker.SendRequest(req)
	//fmt.Println("is running = ", string(body))
	b, err := strconv.ParseBool(string(body))
	return b
}

func (c *SensorCommand) ListSensors(cred *broker.BrokerCredentials, service string) {
	sensors := c.getSensors(cred, service)
	fmt.Println(terminal.ColorizeBold(service, 32))
	for i := 0; i < len(service); i++ {
		fmt.Print(terminal.ColorizeBold("-", 32))
	} 
	fmt.Println()
	c.outputSensorChildren(0, sensors)
}

func (c *SensorCommand) outputSensorChildren(indent int, sensors map[string]interface{}){
	for k, v := range sensors {	
		c.printIndent(indent)
		if indent == 0{
			fmt.Print(terminal.ColorizeBold("Entity:", 32))
		}
		fmt.Println(terminal.ColorizeBold(k, 32))
		c.outputSensors(indent + 1, v.(map[string]interface{}))
	}
}

func (c *SensorCommand) outputSensors(indent int, sensors map[string]interface{}){
	children := sensors["children"]
	for k, v := range sensors {
		if k != "children" {
			c.printIndent(indent)
			switch v.(type) {
				default:
					fmt.Println(k,":", v)
				case map[string]interface{}:
				    fmt.Println(k)
					c.outputSensors(indent + 1, v.(map[string]interface{}))
			}
		}
	}
	if children != nil {
		c.outputSensorChildren(indent + 1, children.(map[string]interface{}))
	}
}

func (c *SensorCommand) printIndent(indent int){
	for i := 0; i < indent; i++ {
		fmt.Print("  ")
	}
}



