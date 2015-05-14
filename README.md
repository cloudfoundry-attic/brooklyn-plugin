# Brooklyn Cloud Foundry Plugin

This project makes a plugin to the cloud foundry [CLI]{https://github.com/cloudfoundry-community/brooklyn-service-broker} to manage services
brokered by the Brooklyn Service Broker.

## Quick Start

If you are using CLI version 6.10+ you can install the 
plugin using Plugin Discovery with the community repository:

    $ cf add-plugin-repo community http://plugins.cloudfoundry.org/
    $ cf install-plugin Brooklyn -r community

Otherwise, you can [build it from source]{docs/build-and-test.md}.  Then login using

    $ cf brooklyn login

which will prompt for a broker, and if not already stored a username and password.
It will then store these details in $HOME/.cf_brooklyn_plugin

The plugin is then ready for [use]{docs/use.md}. 
