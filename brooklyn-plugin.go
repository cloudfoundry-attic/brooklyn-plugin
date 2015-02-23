package main

import (
	
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
    "github.com/cloudfoundry-community/brooklyn-plugin/push"
    "github.com/cloudfoundry-community/brooklyn-plugin/catalog"
	"github.com/cloudfoundry-community/brooklyn-plugin/effectors"
	"github.com/cloudfoundry-community/brooklyn-plugin/sensors"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"fmt"
	"github.com/cloudfoundry/cli/generic"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/cf/terminal"
	"os"
)

type BrooklynPlugin struct{
	ui            terminal.UI
	cliConnection plugin.CliConnection
	yamlMap       generic.Map
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
		push.NewPushCommand(cliConnection).Push(args[1:])
		//c.push(args[1:])
	case "add-catalog":
		assert.Condition(len(args) == 6, "incorrect number of arguments")
		catalog.NewAddCatalogCommand(cliConnection).AddCatalog(broker.NewBrokerCredentials(args[2], args[3], args[4]), args[5])
		defer fmt.Println("Catalog item sucessfully added.")
	case "delete-catalog":
		assert.Condition(len(args) == 7, "incorrect number of arguments")
		catalog.NewAddCatalogCommand(cliConnection).DeleteCatalog(broker.NewBrokerCredentials(args[2], args[3], args[4]), args[5], args[6])
	case "effectors":
		assert.Condition(len(args) == 6, "incorrect number of arguments")
		effectors.NewEffectorCommand(cliConnection).ListEffectors(broker.NewBrokerCredentials(args[2], args[3], args[4]), args[5])
	case "invoke":
		assert.Condition(len(args) >= 7, "incorrect number of arguments")
		effectors.NewEffectorCommand(cliConnection).InvokeEffector(broker.NewBrokerCredentials(args[2], args[3], args[4]), args[5], args[6], args[7:])
	case "sensors":
		assert.Condition(len(args) == 6, "incorrect number of arguments")
		sensors.NewSensorCommand(cliConnection).ListSensors(broker.NewBrokerCredentials(args[2], args[3], args[4]), args[5])
	case "ready":
	    assert.Condition(len(args) == 6, "incorrect number of arguments")
	    fmt.Println("Ready:", sensors.NewSensorCommand(cliConnection).IsServiceReady(broker.NewBrokerCredentials(args[2], args[3], args[4]), args[5]))
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
