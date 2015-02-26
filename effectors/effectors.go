package effectors

import(
	
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"bytes"
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
	"net/http"
	"encoding/json"
	"github.com/cloudfoundry/cli/cf/terminal"
	"strings"
)

type EffectorCommand struct {
	cliConnection plugin.CliConnection
	ui            terminal.UI
}

func NewEffectorCommand(cliConnection plugin.CliConnection, ui terminal.UI) *EffectorCommand{
	command := new(EffectorCommand)
	command.cliConnection = cliConnection
	command.ui = ui
	return command
}

func (c *EffectorCommand) InvokeEffector(cred *broker.BrokerCredentials, service, effector string, params []string) {
	guid, err := c.cliConnection.CliCommandWithoutTerminalOutput("service", service, "--guid")
	assert.ErrorIsNil(err)
	assert.Condition(strings.Contains(effector, ":"), "invalid effector format")
	split := strings.Split(effector, ":")
	path := "invoke/" + guid[0] + "/" + split[0] + "/" + split[1]
	fmt.Println("Invoking effector", terminal.ColorizeBold(effector, 36))
	
	m := make(map[string]string)
	for i := 0; i < len(params); i = i + 2 {
		assert.Condition(strings.HasPrefix(params[i], "--"), "invalid parameter format")
		k := strings.TrimPrefix(params[i], "--")
		v := params[i + 1]
		
		m[k] = v
	}
	post, err := json.Marshal(m)
	assert.ErrorIsNil(err)
	req, err := http.NewRequest("POST", broker.CreateRestCallUrlString(c.cliConnection, cred, path), bytes.NewBuffer(post))
	req.Header.Set("Content-Type", "application/json")
	assert.ErrorIsNil(err)
	body, _ := broker.SendRequest(req)
	fmt.Println(string(body))
}

func (c *EffectorCommand) ListEffectors(cred *broker.BrokerCredentials, service string) {
	guid, err := c.cliConnection.CliCommandWithoutTerminalOutput("service", service, "--guid")
	url := broker.CreateRestCallUrlString(c.cliConnection, cred, "effectors/" + guid[0])
	req, err := http.NewRequest("GET", url, nil)
	assert.ErrorIsNil(err)
	body, _ := broker.SendRequest(req)
	//fmt.Println(string(body))
	var effectors map[string]interface{}
	err = json.Unmarshal(body, &effectors)
	assert.ErrorIsNil(err)
	fmt.Println(terminal.ColorizeBold(service, 32))
	for i := 0; i < len(service); i++ {
		fmt.Print(terminal.ColorizeBold("-", 32))
	} 
	fmt.Println()
	c.outputChildren(0, effectors)
	
}

func (c *EffectorCommand) outputChildren(indent int, effectors map[string]interface{}){
	children := effectors["children"]
	for k, v := range effectors {	
		if k != "children" {
			c.printIndent(indent)
			if indent == 0{
				fmt.Print(terminal.ColorizeBold("Application:", 32))
			}
			fmt.Println(terminal.ColorizeBold(k, 32))
			c.outputEffectors(indent + 1, v.(map[string]interface{}))
		}
	}
	
	if children != nil {
		c.outputChildren(indent + 1, children.(map[string]interface{}))
	}
}

func (c *EffectorCommand) outputEffectors(indent int, effectors map[string]interface{}){
	children := effectors["children"]
	for k, v := range effectors {
		if k != "children" {
			c.printIndent(indent)
			c.printEffectorDescription(indent, terminal.ColorizeBold(k, 31), v.(map[string]interface{}))
		}
	}
	if children != nil {
		c.outputChildren(indent, children.(map[string]interface{}))
	}
}

func (c *EffectorCommand) printEffectorDescription(indent int, effectorName string,  effector map[string]interface{}){
	params := effector["parameters"].([]interface {})
	
	fmt.Printf("%-30s %s\n", effectorName, effector["description"].(string))
	
	if len(params) != 0 {
		
		c.printIndent(indent + 1)
		fmt.Println("parameters: ")
		for _, k := range params {
			c.printParameterDescription(indent + 1, k.(map[string]interface{}))
		}
	}
	
}

func (c *EffectorCommand) printParameterDescription(indent int, parameter map[string]interface{}) {
	
	c.printIndent(indent)
	fmt.Printf("%-17s %-s\n", parameter["name"].(string), parameter["description"].(string))
}

func (c *EffectorCommand) printIndent(indent int){
	for i := 0; i < indent; i++ {
		fmt.Print("  ")
	}
}

