// add-catalog
package main

import(
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
	"net/http"
	"path/filepath"
	"os"
)

type AddCatalogCommand struct {
	cliConnection plugin.CliConnection
}

func NewAddCatalogCommand(cliConnection plugin.CliConnection) *AddCatalogCommand{
	command := new(AddCatalogCommand)
	command.cliConnection = cliConnection
	return command
}

func (c *AddCatalogCommand) addCatalog(cred *BrokerCredentials, filePath string) {
	fmt.Println("Adding Brooklyn catalog item...")
	
	file, err := os.Open(filepath.Clean(filePath))
	AssertErrorIsNil(err)
	defer file.Close()
	
	req, err := http.NewRequest("POST", CreateRestCallUrlString(c.cliConnection, cred, "create"), file)
	AssertErrorIsNil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	SendRequest(req)
}

func (c *AddCatalogCommand) deleteCatalog(cred *BrokerCredentials, name, version string) {
	fmt.Println("Deleting Brooklyn catalog item...")
	req, err := http.NewRequest("DELETE", 
	    CreateRestCallUrlString(c.cliConnection, cred, "delete/" +name+ "/" + version + "/"), 
		nil)
	AssertErrorIsNil(err)
	SendRequest(req)
}
