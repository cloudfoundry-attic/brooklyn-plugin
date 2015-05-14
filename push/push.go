package push

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"github.com/cloudfoundry-community/brooklyn-plugin/catalog"
	"github.com/cloudfoundry-community/brooklyn-plugin/io"
	"github.com/cloudfoundry-community/brooklyn-plugin/sensors"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/generic"
	"github.com/cloudfoundry/cli/plugin"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PushCommand struct {
	cliConnection plugin.CliConnection
	ui            terminal.UI
	yamlMap       generic.Map
	credentials   *broker.BrokerCredentials
}

func NewPushCommand(cliConnection plugin.CliConnection, ui terminal.UI, credentials *broker.BrokerCredentials) *PushCommand {
	command := new(PushCommand)
	command.cliConnection = cliConnection
	command.ui = ui
	command.credentials = credentials
	return command
}

/*
   modify the application manifest before passing to to original command
*/
func (c *PushCommand) Push(args []string) {
	// args[0] == "push"

	// TODO use CF way of parsing args
	manifest := "manifest.yml"
	if len(args) >= 3 && args[1] == "-f" {
		manifest = args[2]
		args = append(args[:1], args[3:]...)
	}
	c.yamlMap = io.ReadYAMLFile(manifest)

	//fmt.Println("getting brooklyn")
	allCreatedServices := []string{}
	allCreatedServices = append(allCreatedServices, c.replaceTopLevelServices()...)
	allCreatedServices = append(allCreatedServices, c.replaceApplicationServices()...)
	
	for _, service := range allCreatedServices {
		fmt.Printf("Waiting for %s to start...\n", service)
	}

	c.waitForServiceReady(allCreatedServices)

	c.pushWith(args, "manifest.temp.yml")
}

func (c *PushCommand) waitForServiceReady(services []string) {
	// before pushing check to see if service is running

	ready := c.allReady(services)
	waitTime := 2 * time.Second
	for !ready {
		fmt.Printf("Trying again in %v\n", waitTime)
		time.Sleep(waitTime)
		ready = c.allReady(services)
		if 2*waitTime == 16*time.Second {
			waitTime = 15 * time.Second
		} else if 2*waitTime > time.Minute {
			waitTime = time.Minute
		} else {
			waitTime = 2 * waitTime
		}
	}
}

func (c *PushCommand) allReady(services []string) bool {
	ready := true
	for _, service := range services {
		serviceReady := sensors.NewSensorCommand(c.cliConnection, c.ui).IsServiceReady(c.credentials, service)
		if !serviceReady {
			fmt.Printf("%s is not yet running.\n", service)
		}
		ready = ready && serviceReady
	}
	return ready
}

func (c *PushCommand) pushWith(args []string, tempFile string) {
	io.WriteYAMLFile(c.yamlMap, tempFile)
	_, err := c.cliConnection.CliCommand(append(args, "-f", tempFile)...)
	assert.ErrorIsNil(err)
	err = os.Remove(tempFile)
	assert.ErrorIsNil(err)
}

func (c *PushCommand) replaceTopLevelServices() []string{
	allCreatedServices := []string{}
	services := c.yamlMap.Get("services").([]interface{})
	for i, service := range services {
		switch service.(type) {
		case string: // do nothing, since service is an existing named service
		case map[interface{}]interface{}:
		   	createdService := c.newServiceFromMap(service.(map[interface{}]interface{}))
		   	allCreatedServices = append(allCreatedServices, createdService)
			// replace the defn in the yaml for its name
			services[i] = createdService
		}
	}
	return allCreatedServices
}

func (c *PushCommand) replaceApplicationServices() []string {
	allCreatedServices := []string{}
	applications := c.yamlMap.Get("applications").([]interface{})
	for _, app := range applications {
		application, found := app.(map[interface{}]interface{})
		assert.Condition(found, "Application not found.")
		allCreatedServices = append(allCreatedServices, c.replaceBrooklynCreatingServices(application)...)
		allCreatedServices = append(allCreatedServices, c.replaceServicesCreatingServices(application)...)
	}
	
	return allCreatedServices
}

func (c *PushCommand) replaceBrooklynCreatingServices(application map[interface{}]interface{}) []string {
	brooklyn, found := application["brooklyn"].([]interface{})
	var createdServices []string
	if !found { return createdServices }
	createdServices = c.createAllServicesFromBrooklyn(brooklyn)
	application["services"] = c.mergeServices(application, createdServices)
	delete(application, "brooklyn")
	return createdServices
}

func (c *PushCommand) replaceServicesCreatingServices(application map[interface{}]interface{}) []string {
	services, found := application["services"].([]interface{})
	createdServices := []string{}
	if !found { return createdServices }
	createdServices = c.createAllServicesFromServices(services)
	return createdServices
}

func (c *PushCommand) mergeServices(application map[interface{}]interface{}, services []string) []string {
	if oldServices, found := application["services"].([]interface{}); found {
		for _, name := range oldServices {
			services = append(services, name.(string))
		}
	}
	return services
}

func (c *PushCommand) createAllServicesFromServices(services []interface{}) []string {
	var createdServices []string
	for i, service := range services {
		switch service.(type) {
		case string: // do nothing, since service is an existing named service
		case map[interface{}]interface{}:
		   	// service definition
			createdService := c.newServiceFromMap(service.(map[interface{}]interface{}))
		   	createdServices = append(createdServices, createdService)
			services[i] = createdService
		}
	}
	return createdServices
}

func (c *PushCommand) createAllServicesFromBrooklyn(brooklyn []interface{}) []string {
	services := []string{}
	for _, brooklynApp := range brooklyn {
		brooklynApplication, found := brooklynApp.(map[interface{}]interface{})
		assert.Condition(found, "Expected Map.")
		services = append(services, c.newService(brooklynApplication))
	}
	return services
}

func (c *PushCommand) newServiceFromMap(service map[interface{}]interface{}) string {
	name, found := service["name"].(string)
	assert.Condition(found, "no name specified in blueprint")
	location, found := service["location"].(string)
	assert.Condition(found, "no location specified")
	if exists := c.catalogItemExists(name); !exists {
		c.createNewCatalogItemWithoutLocation(name, []interface{}{service})
	}
	c.cliConnection.CliCommand("create-service", name, location, name)
	return name
}

// expects an item from the brooklyn section with a name section
func (c *PushCommand) newService(brooklynApplication map[interface{}]interface{}) string {
	name, found := brooklynApplication["name"].(string)
	assert.Condition(found, "Expected Name.")
	c.createServices(brooklynApplication, name)
	return name
}

// expects an item from the brooklyn section
// 
func (c *PushCommand) createServices(brooklynApplication map[interface{}]interface{}, name string) {
	// If there is a service section then this refers to an
	// existing catalog entry.
	service, found := brooklynApplication["service"].(string)
	if found {
		// now we must use an existing plan (location)
		location, found := brooklynApplication["location"].(string)
		assert.Condition(found, "Expected Location")
		c.cliConnection.CliCommand("create-service", service, location, name)
	} else {
		c.extractAndCreateService(brooklynApplication, name)
	}
}

func (c *PushCommand) extractAndCreateService(brooklynApplication map[interface{}]interface{}, name string) {
	// If there is a services section then this is a blueprint
	// and this should be extracted and sent as a catalog item
	blueprints, found := brooklynApplication["services"].([]interface{})
	var location string
	if found {

		// only do this if catalog doesn't contain it already
		// now we decide whether to add a location to the
		// catalog item, or use all locations as plans
		switch brooklynApplication["location"].(type) {
		case string:
			location = brooklynApplication["location"].(string)
			if exists := c.catalogItemExists(name); !exists {
				c.createNewCatalogItemWithoutLocation(name, blueprints)
			}
		case map[interface{}]interface{}:
			locationMap := brooklynApplication["location"].(map[interface{}]interface{})
			count := 0
			for key, _ := range locationMap {
				location, found = key.(string)
				assert.Condition(found, "location not found")
				count = count + 1
			}
			assert.Condition(count == 1, "Expected only one location")
			if exists := c.catalogItemExists(name); !exists {
				c.createNewCatalogItemWithLocation(name, blueprints, locationMap)
			}
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

func (c *PushCommand) createCatalogYamlMap(name string, blueprintMap []interface{}) generic.Map {
	yamlMap := generic.NewMap()
	entry := map[string]string{
		"id":          name,
		"version":     "1.0",
		"iconUrl":     "",
		"description": "A user defined blueprint",
	}
	yamlMap.Set("brooklyn.catalog", entry)
	yamlMap.Set("name", name)
	yamlMap.Set("services", []map[string]interface{}{
		map[string]interface{}{
			"type":              "brooklyn.entity.basic.BasicApplication",
			"name":              name,
			"brooklyn.children": blueprintMap,
		},
	})
	return yamlMap
}

func (c *PushCommand) createNewCatalogItemWithLocation(
	name string, blueprintMap []interface{}, location map[interface{}]interface{}) {
	yamlMap := c.createCatalogYamlMap(name, blueprintMap)
	yamlMap.Set("location", generic.NewMap(location))
	c.createNewCatalogItem(name, yamlMap)
}

func (c *PushCommand) createNewCatalogItemWithoutLocation(name string, blueprintMap []interface{}) {
	yamlMap := c.createCatalogYamlMap(name, blueprintMap)
	c.createNewCatalogItem(name, yamlMap)
}

func (c *PushCommand) createNewCatalogItem(name string, yamlMap generic.Map) {
	tempFile := "catalog.temp.yml"
	io.WriteYAMLFile(yamlMap, tempFile)

	cred := c.credentials
	brokerUrl, err := broker.ServiceBrokerUrl(c.cliConnection, cred.Broker)
	assert.ErrorIsNil(err)

	catalog.NewAddCatalogCommand(c.cliConnection, c.ui).AddCatalog(cred, tempFile)

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

func (c *PushCommand) randomString(size int) string {
	rb := make([]byte, size)
	_, err := rand.Read(rb)
	assert.ErrorIsNil(err)
	return base64.URLEncoding.EncodeToString(rb)
}
