package push

import (
	
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"github.com/cloudfoundry-community/brooklyn-plugin/io"
	"fmt"
	"path/filepath"
	"net/http"
	//"github.com/cloudfoundry/cli/cf/errors"
	//. "github.com/cloudfoundry/cli/cf/i18n"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/generic"
	"os"
	"strings"
	//"github.com/cloudfoundry-incubator/candiedyaml"
	"encoding/base64"
    "crypto/rand"
	//"io"
)

type PushCommand struct {
	cliConnection plugin.CliConnection
	ui            terminal.UI
	yamlMap       generic.Map
	credentials   *broker.BrokerCredentials
}

func NewPushCommand(cliConnection plugin.CliConnection, ui terminal.UI) *PushCommand{
	command := new(PushCommand)
	command.cliConnection = cliConnection
	command.ui = ui
	return command
}

/*
    modify the application manifest before passing to to original command
    TODO: We need to ensure that multiple calls to push do not keep 
	      instantiating new instances of services that are already running
*/
func (c *PushCommand) Push(args []string){
	fmt.Println("Running the brooklyn command")
	// TODO if -f flag sets manifest use that instead
		
	c.yamlMap = io.ReadYAMLFile("manifest.yml")
	
	fmt.Println("getting brooklyn")
	applications := c.yamlMap.Get("applications").([]interface{})
	for _, app := range applications {
		//fmt.Println("app...\n", app)
		application, found := app.(map[interface{}]interface{})
		assert.Condition(found, "Application not found.")
		c.replaceBrooklynCreatingServices(application)
	}
	// before pushing check to see if service is running
	//creds := c.promptForBrokerCredentials()
	//ready := sensors.NewSensorCommand(cliConnection).IsServiceReady(creds, args[5]);
	//timeout := 2 * time.Second
	//for !ready {
	//	time.Sleep(timeout)
		
	//}
	
	c.pushWith(args, "manifest.temp.yml")
}

func (c *PushCommand) pushWith(args []string, tempFile string) {
	io.WriteYAMLFile(c.yamlMap, tempFile)
	_, err := c.cliConnection.CliCommand(append(args, "-f", tempFile)...)
	assert.ErrorIsNil(err)
	err = os.Remove(tempFile)
	assert.ErrorIsNil(err)
}

func (c *PushCommand) replaceBrooklynCreatingServices(application map[interface{}]interface{}){
	brooklyn, found := application["brooklyn"].([]interface{})
	assert.Condition(found, "Brooklyn not found.")
	// check to see if services section already exists
	application["services"] = c.mergeServices(application, c.createAllServices(brooklyn))
	//fmt.Println("\nmodified...", application)
	delete(application, "brooklyn")
	//fmt.Println("\nmodified...", application)
}

func (c *PushCommand) mergeServices(application map[interface{}]interface{}, services []string) []string {
	if oldServices, found := application["services"].([]interface {}); found {
		for _, name := range oldServices {
			//fmt.Println("found", name)
    		services = append(services, name.(string))
		}
	}
	return services
}

func (c *PushCommand) createAllServices(brooklyn []interface{}) []string{
	services := []string{}
	for _, brooklynApp := range brooklyn {
		//fmt.Println("brooklyn app... \n", brooklynApp)
		brooklynApplication, found := brooklynApp.(map[interface{}]interface{})
		assert.Condition(found, "Expected Map.")
		services = append(services, c.newService(brooklynApplication))	
	}
	//fmt.Println("finished creating services \n")
	return services
}

func (c *PushCommand) newService(brooklynApplication map[interface{}]interface{}) string{
	name, found := brooklynApplication["name"].(string)
	assert.Condition(found, "Expected Name.")
	location, found := brooklynApplication["location"].(string)
	assert.Condition(found, "Expected Location")
	//fmt.Println("creating service:",name, location)
	c.createServices(brooklynApplication, name, location)
	return name
}

func (c *PushCommand) createServices(brooklynApplication map[interface{}]interface{}, name, location string){
	// If there is a service section then this refers to an
	// existing catalog entry.
	service, found := brooklynApplication["service"].(string)
	if found {
		c.cliConnection.CliCommand("create-service", service, location, name)
	} else {
		c.extractAndCreateService(brooklynApplication, name, location)
	}
}

func (c *PushCommand) extractAndCreateService(brooklynApplication map[interface{}]interface{}, name, location string){
	// If there is a services section then this is a blueprint
	// and this should be extracted and sent as a catalog item 
	blueprints, found := brooklynApplication["services"].([]interface{})
	
	// only do this if catalog doesn't contain it already
	if found {
		//fmt.Println("found catalog entry")
		if exists := c.catalogItemExists(name); !exists {
			c.createNewCatalogItem(name, blueprints)
		}
		c.cliConnection.CliCommand("create-service", name, location, name)
	}
}

func (c *PushCommand) catalogItemExists(name string) bool {
	services, err := c.cliConnection.CliCommandWithoutTerminalOutput("marketplace", "-s", name)
	if err != nil {
		return false
	}
	
	for _, a := range services {
		fields := strings.Fields(a)
		if fields[0] == "OK" {
			return true
		}
	}
	return false
}

func (c *PushCommand) createNewCatalogItem(name string, blueprintMap []interface{}){
	yamlMap := generic.NewMap()
	entry := map[string]string{
    	"id": c.randomString(8),
    	"version": "1.0",
    	"iconUrl": "",
    	"description": "A user defined blueprint",
	}
	yamlMap.Set("brooklyn.catalog", entry)
	yamlMap.Set("name", name)
	yamlMap.Set("services", []map[string]interface{}{
		map[string]interface{}{
			"type": "brooklyn.entity.basic.BasicApplication",
			"brooklyn.children": blueprintMap,
		},
	})
	tempFile := "catalog.temp.yml"
	io.WriteYAMLFile(yamlMap, tempFile)
	
	
	cred := c.promptForBrokerCredentials()
	brokerUrl, err := broker.ServiceBrokerUrl(c.cliConnection, cred.Broker)
	assert.ErrorIsNil(err)
	//fmt.Println(brokerUrl)
	c.addCatalog(cred, tempFile)

	c.cliConnection.CliCommand("update-service-broker", cred.Broker, cred.Username, cred.Password, brokerUrl)
	c.cliConnection.CliCommand("enable-service-access", name)
	err = os.Remove(tempFile)
	assert.ErrorIsNil(err)
}

func (c *PushCommand) addCatalog(cred *broker.BrokerCredentials, filePath string) {
	fmt.Println("Adding Brooklyn catalog item...")
	
	file, err := os.Open(filepath.Clean(filePath))
	assert.ErrorIsNil(err)
	defer file.Close()
	
	req, err := http.NewRequest("POST", broker.CreateRestCallUrlString(c.cliConnection, cred, "create"), file)
	assert.ErrorIsNil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	broker.SendRequest(req)
}


func (c *PushCommand) promptForBrokerCredentials() *broker.BrokerCredentials{
	if c.credentials != nil {
		return c.credentials
	}
	brokerName := c.ui.Ask("Broker")
	username := c.ui.Ask("Username")
	password := c.ui.AskForPassword("Password")
	c.credentials = broker.NewBrokerCredentials(brokerName, username, password)
	return c.credentials
}

func (c *PushCommand) randomString(size int) string{
	rb := make([]byte,size)
  	_, err := rand.Read(rb)
	assert.ErrorIsNil(err)
	return base64.URLEncoding.EncodeToString(rb)
}

