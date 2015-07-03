package push

import (
	"fmt"
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"github.com/cloudfoundry-community/brooklyn-plugin/io"
	"github.com/cloudfoundry-community/brooklyn-plugin/sensors"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/generic"
	"github.com/cloudfoundry/cli/plugin"
	"encoding/json"
	"os"
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

func (c *PushCommand) replaceTopLevelServices() []string {
	allCreatedServices := []string{}
	services, found := c.yamlMap.Get("services").([]interface{})
	if !found {
		return allCreatedServices
	}
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
	applications, found := c.yamlMap.Get("applications").([]interface{})
	if !found {
		return allCreatedServices
	}
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
	if !found {
		return createdServices
	}
	createdServices = c.createAllServicesFromBrooklyn(brooklyn)
	application["services"] = c.mergeServices(application, createdServices)
	delete(application, "brooklyn")
	return createdServices
}

func (c *PushCommand) replaceServicesCreatingServices(application map[interface{}]interface{}) []string {
	services, found := application["services"].([]interface{})
	createdServices := []string{}
	if !found {
		return createdServices
	}
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
	_, found = service["location"].(string)
	assert.Condition(found, "no location specified")
	c.createServices(service, name)
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
	// we expect this brooklynApplication map to be a brooklyn blueprint
	
	var data map[string]interface{}
	data = JsonSafeMap(brooklynApplication)
	fmt.Printf("Marshalling brooklynApplication: %v\n", brooklynApplication)
	
	fmt.Printf("created map: %v\n", data)
	jsonString, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("'%v'\n",string(jsonString))
	c.cliConnection.CliCommand("create-service", "User Defined", "blueprint", name, "-c", fmt.Sprintf("%v",string(jsonString)))
}

func JsonSafeArray(arr []interface{}) []interface{} {
	var data []interface{}
	for _, value := range arr {
		switch value.(type) {
			case string: data = append(data, value.(string))
			case []interface{} : data = append(data, JsonSafeArray(value.([]interface{})))
			case map[interface{}]interface{} : data = append(data, JsonSafeMap(value.(map[interface{}]interface{})))
		}
	}
	return data
}

func JsonSafeMap(m map[interface{}]interface{}) map[string]interface{} {
	var data map[string]interface{}
	data = make(map[string]interface{})
	for key, value := range m {
		newKey, found := key.(string)
		assert.Condition(found, "Expected a string key")
		switch value.(type) {
			case string: data[newKey] = value.(string)
			case []interface{}: data[newKey] = JsonSafeArray(value.([]interface{}))
			case map[interface{}]interface{}: data[newKey] = JsonSafeMap(value.(map[interface{}]interface{}))
		}
	}
	return data
}