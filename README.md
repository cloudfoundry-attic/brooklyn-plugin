Brooklyn Cloud Foundry Plugin
-----------------------------
To build,

    $ go build

To install,

    $ cf install-plugin brooklyn-plugin

Login
-----

    $ cf brooklyn login

this will prompt for a broker, and if not already stored a username and password.
It will then store these details in $HOME/.cf_brooklyn_plugin


Push
----

    $ cf brooklyn push

this will lookup the manifest.yml, and if it contains a section
called brooklyn, it will use a service broker to create these
services and create a new manifest.temp.yml file taking out
the brooklyn service and replacing it with a services section
containing the service instances created by the brooklyn service
broker. It will then delegate to the original push command with
the manifest.temp.yml file before deleting it.

For example,

    applications:
    - name: my-app
      ...
    brooklyn:
    - name: my-demo-web-cluster
      location: localhost
      service: Demo Web Cluster with DB
    services:
    - old-service

creates an instance of the service `Demo Web Cluster with DB` in the service broker with the plan `localhost` before creating the temp file,

    applications:
    - name: my-app
      ...
    services:
    - my-demo-web-cluster
    - old-service

to use with push.

It is also possible to specify a Brooklyn blueprint in the manifest under the brooklyn section:

    applications:
    - name: my-app
    brooklyn:
    - name: my-MySQL
      location: localhost
      services: 
      - type: brooklyn.entity.database.mysql.MySqlNode
    services:
    - old-service

In this instance, the brooklyn section will be extracted and converted into a catalog.temp.yml file:

    brooklyn.catalog:
        id: <randomly-generated-id>
        version: 1.0
        iconUrl: 
        description: A user defined blueprint 
    name: my-MySQL
    services:
    - type: brooklyn.entity.basic.BasicApplication
      brooklyn.children:
      - type: brooklyn.entity.database.mysql.MySqlNode
      
The user is then prompted for a broker with its username and password for which to submit this.  The broker will then be refreshed and the service enabled.  Then the service broker will create an instance of this service and replace the section in the manifest with,

    applications:
    - name: my-app
      ...
    services:
    - my-MySQL
    - old-service
	
Push will then wait for the service to be provisioned before delegating to the original push for binding etc.

Adding catalog items manually
-----------------------------

    $ cf brooklyn add-catalog [<broker> <username> <password>] <path/to/blueprint.yml>
    
this allows new entities to be created and added to the brooklyn
catalog.  The service broker that is associated will need to be
refreshed with `cf update-service-broker` and enabled with 
`enable-service-access` for these new services to become available.

Deleting catalog items
----------------------

    $ cf brooklyn delete-catalog [<broker> <username> <password>] <name> <version>
    
this allows catalog items to be deleted from the service broker.
As with `add-catalog`, the service broker will need to be refreshed
for the changes to take effect.

Listing Effectors
-----------------

    $ cf brooklyn effectors [<broker> <username> <password>] <service>
    
this lists all of the effectors that can be invoked on the specified service.


Invoking Effectors
------------------

    $ cf brooklyn invoke [<broker> <username> <password>] <service> <effector>
	
invokes the effector on this service.
    
Viewing Sensors
---------------

    $ cf brooklyn sensors [<broker> <username> <password>] <service>
	
views the sensors associated with this service.

Check if a service is ready for binding
---------------------------------------

    $ cf brooklyn ready [<broker> <username> <password>] <service>
	
checks if the service has been provisioned yet and is running.
It is useful for this to be true before binding, since the
VCAP_SERVICES variable will contain the sensor information that
exists at bind time.

Uninstall
---------

    $ cf uninstall-plugin BrooklynPlugin
