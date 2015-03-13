package catalog

import (
	"fmt"
	"github.com/cloudfoundry-community/brooklyn-plugin/assert"
	"github.com/cloudfoundry-community/brooklyn-plugin/broker"
	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/plugin"
	"net/http"
	"os"
	"path/filepath"
)

type AddCatalogCommand struct {
	cliConnection plugin.CliConnection
	ui            terminal.UI
}

func NewAddCatalogCommand(cliConnection plugin.CliConnection, ui terminal.UI) *AddCatalogCommand {
	command := new(AddCatalogCommand)
	command.cliConnection = cliConnection
	command.ui = ui
	return command
}

func (c *AddCatalogCommand) AddCatalog(cred *broker.BrokerCredentials, filePath string) {
	fmt.Println("Adding Brooklyn catalog item...")

	file, err := os.Open(filepath.Clean(filePath))
	assert.ErrorIsNil(err)
	defer file.Close()

	req, err := http.NewRequest("POST", broker.CreateRestCallUrlString(c.cliConnection, cred, "create"), file)
	assert.ErrorIsNil(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	broker.SendRequest(req)
}

func (c *AddCatalogCommand) DeleteCatalog(cred *broker.BrokerCredentials, name, version string) {
	fmt.Println("Deleting Brooklyn catalog item...")
	req, err := http.NewRequest("DELETE",
		broker.CreateRestCallUrlString(c.cliConnection, cred, "delete/"+name+"/"+version+"/"),
		nil)
	assert.ErrorIsNil(err)
	broker.SendRequest(req)
}
