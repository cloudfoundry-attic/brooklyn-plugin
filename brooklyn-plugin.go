package main

import (
	"path/filepath"
	"fmt"
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"github.com/cloudfoundry-community/brooklyn-plugin/catalog"
	"github.com/cloudfoundry-community/brooklyn-plugin/effectors"
	"github.com/cloudfoundry-community/brooklyn-plugin/io"
	"github.com/cloudfoundry-community/brooklyn-plugin/push"
	"github.com/cloudfoundry-community/brooklyn-plugin/sensors"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/generic"
	"github.com/cloudfoundry/cli/plugin"
	"os"
)

type BrooklynPlugin struct {
	ui            terminal.UI
	cliConnection plugin.CliConnection
	yamlMap       generic.Map
	credentials   *broker.BrokerCredentials
}

func (c *BrooklynPlugin) printHelp(name string) {
	metadata := c.GetMetadata()
	for _, command := range metadata.Commands {
		if command.Name == name{
			fmt.Println("Name:")
			fmt.Printf("    %-s - %-s\n", command.Name, command.HelpText)
			fmt.Println("Usage:")
			fmt.Printf("    %-s\n", command.UsageDetails.Usage)
			return
		}
	}
}

func (c *BrooklynPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	argLength := len(args)

	c.ui = terminal.NewUI(os.Stdin, terminal.NewTeePrinter())
	c.cliConnection = cliConnection
	
	if argLength == 1 {
		metadata := c.GetMetadata()
		for _, command := range metadata.Commands {
			fmt.Printf("%-25s %-50s\n", command.Name, command.HelpText)
		}
		return
	}
	
	if argLength == 3 && args[2] == "-h" {
		c.printHelp(args[0] + " " + args[1])
		return
	}

	// check to see if ~/.cf_brooklyn_plugin exists
	// Then Parse it to get user credentials
	home := os.Getenv("HOME")
	file := filepath.Join(home, ".cf_brooklyn_plugin")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		io.WriteYAMLFile(generic.NewMap(), file)
	}
	yamlMap := io.ReadYAMLFile(file)
	var target, username, password string
	target, found := yamlMap.Get("target").(string)
	if found {
		auth, found := yamlMap.Get("auth").(map[interface{}]interface{})
		if found {
			creds := auth[target].(map[interface{}]interface{})
			username, found = creds["username"].(string)
			if found {
				password, found = creds["password"].(string)
			}
		}
	}

	brokerCredentials := broker.NewBrokerCredentials(target, username, password)

	switch args[1] {
	case "login":
		broker := c.ui.Ask("Broker")
		if !yamlMap.Has("auth"){
			yamlMap.Set("auth", generic.NewMap())
		}
		auth := generic.NewMap(yamlMap.Get("auth"))
		
		if !auth.Has(broker) {
			user := c.ui.Ask("Username")
			pass := c.ui.AskForPassword("Password")
			auth.Set(broker, generic.NewMap(map[string]string{
				"username": user,
				"password": pass,
			}))
		}
		yamlMap.Set("target", broker)
		io.WriteYAMLFile(yamlMap, file)
	case "push":
		push.NewPushCommand(cliConnection, c.ui, brokerCredentials).Push(args[1:])
	case "add-catalog":
		if argLength == 3 {
			assert.Condition(found, "target not set")
			catalog.NewAddCatalogCommand(cliConnection, c.ui).AddCatalog(brokerCredentials, args[2])
		} else if argLength == 6 {
			brokerCredentials = broker.NewBrokerCredentials(args[2], args[3], args[4])
			catalog.NewAddCatalogCommand(cliConnection, c.ui).AddCatalog(brokerCredentials, args[5])
		} else {
			assert.Condition(false, "incorrect number of arguments")
		}
		defer fmt.Println("Catalog item sucessfully added.")
	case "delete-catalog":
		if argLength == 4 {
			assert.Condition(found, "target not set")
			catalog.NewAddCatalogCommand(cliConnection, c.ui).DeleteCatalog(brokerCredentials, args[2], args[3])
		} else if argLength == 7 {
			brokerCredentials = broker.NewBrokerCredentials(args[2], args[3], args[4])
			catalog.NewAddCatalogCommand(cliConnection, c.ui).DeleteCatalog(brokerCredentials, args[5], args[6])
		} else {
			assert.Condition(false, "incorrect number of arguments")
		}
	case "effectors":
		if argLength == 3 {
			assert.Condition(found, "target not set")
			effectors.NewEffectorCommand(cliConnection, c.ui).ListEffectors(brokerCredentials, args[2])
		} else if argLength == 6 {
			brokerCredentials = broker.NewBrokerCredentials(args[2], args[3], args[4])
			effectors.NewEffectorCommand(cliConnection, c.ui).ListEffectors(brokerCredentials, args[5])
		} else {
			assert.Condition(false, "incorrect number of arguments")
		}
	case "invoke":
		// TODO need to take a flag to specify broker creds
		// if args[2] == -b then args[2:4] are broker creds
		// so that command can be run without specifying
		// broker credentials
		if argLength >= 7 {
			brokerCredentials = broker.NewBrokerCredentials(args[2], args[3], args[4])
			effectors.NewEffectorCommand(cliConnection, c.ui).InvokeEffector(brokerCredentials, args[5], args[6], args[7:])
		} else {
			assert.Condition(false, "incorrect number of arguments")
		}
	case "sensors":
		if argLength == 3 {
			assert.Condition(found, "target not set")
			sensors.NewSensorCommand(cliConnection, c.ui).ListSensors(brokerCredentials, args[2])
		} else if argLength == 6 {
			brokerCredentials = broker.NewBrokerCredentials(args[2], args[3], args[4])
			sensors.NewSensorCommand(cliConnection, c.ui).ListSensors(brokerCredentials, args[5])
		} else {
			assert.Condition(false, "incorrect number of arguments")
		}
	case "ready":
		if argLength == 3 {
			assert.Condition(found, "target not set")
			fmt.Println("Ready:", sensors.NewSensorCommand(cliConnection, c.ui).IsServiceReady(brokerCredentials, args[2]))
		} else if argLength == 6 {
			brokerCredentials = broker.NewBrokerCredentials(args[2], args[3], args[4])
			fmt.Println("Ready:", sensors.NewSensorCommand(cliConnection, c.ui).IsServiceReady(brokerCredentials, args[5]))
		} else {
			assert.Condition(false, "incorrect number of arguments")
		}
	}
	fmt.Println(terminal.ColorizeBold("OK", 32))

}

func (c *BrooklynPlugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "BrooklynPlugin",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
			Build: 2,
		},
		Commands: []plugin.Command{
			{ // required to be a registered command
				Name:     "brooklyn",
				HelpText: "Brooklyn plugin command's help text",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn",
				},
			},
			{
				Name:     "brooklyn login",
				HelpText: "Store Broker login credentials for use between commands",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn login",
				},
			},
			{
				Name: "brooklyn push",
				HelpText: "Push a new app, replacing " +
					"brooklyn section with instantiated services",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn push [-f MANIFEST]",
				},
			},
			{
				Name: "brooklyn add-catalog",
				HelpText: "Submit a Blueprint to Brooklyn to be " +
					"added to its catalog",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn add-catalog CATALOG",
				},
			},
			{
				Name:     "brooklyn delete-catalog",
				HelpText: "Delete an item from the Brooklyn catalog",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn delete-catalog SERVICE VERSION",
				},
			},
			{
				Name:     "brooklyn effectors",
				HelpText: "List the effectors available to a service",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn effectors [BROKER USERNAME PASSWORD] SERVICE",
				},
			},
			{
				Name:     "brooklyn invoke",
				HelpText: "Invoke an effector on a service",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn invoke [BROKER USERNAME PASSWORD] SERVICE EFFECTOR",
				},
			},
			{
				Name:     "brooklyn sensors",
				HelpText: "List the sensors with their outputs for a service",
				UsageDetails: plugin.Usage{
					Usage: "cf brooklyn sensors [BROKER USERNAME PASSWORD] SERVICE",
				},
			},
		},
	}
}

func main() {
	plugin.Start(new(BrooklynPlugin))
}
