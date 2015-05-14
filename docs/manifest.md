# Adding service definitions to the Application Manifest

When doing a 

    $ cf brooklyn push

this will lookup the manifest.yml and look for service descriptions in 
any of three locations:

1. Under a `brooklyn` section for an application.  This section must contain 
three fields: a name, a location, and a service(s).  These correspond to the 
Name, Plan, and Service from the Brooklyn Service Broker.

2. Under the top-level `services` section. A Brooklyn blueprint that contains 
as a minimum, a name, a location, and a type.

3. Under an application-level `services` section.  This also is a Brooklyn 
blueprint that contains as a minimum, a name, a location, and a type.

The Brooklyn Service Broker will create these services and generate a new 
manifest.temp.yml file taking out the service definitions replacing them in 
a services section containing the service instances created by the broker. 
It will then delegate to the original push command withthe manifest.temp.yml 
file before deleting it.

## Example 1.

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

## Example 2.

Specify a Brooklyn blueprint in the manifest under the brooklyn section:

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
      
The broker will then be refreshed and the service enabled.  Then the 
service broker will create an instance of this service and 
replace the section in the manifest with,

    applications:
    - name: my-app
      ...
      services:
      - my-MySQL
      - old-service

## Example 3.

Top level services.

    applications:
    - name: my-app
      ...
    services:
    - old-service
    - name: my-MySQL
      location: localhost
      type: brooklyn.entity.database.mysql.MySqlNode

## Example 4.

Application-level services

    applications:
    - name: my-app
      services:
      - old-service
      - name: my-MySQL
        location: localhost
        type: brooklyn.entity.database.mysql.MySqlNode    
	
# Wait for service up
The Brooklyn Push command will then wait for the service to be provisioned 
before delegating to the original push for binding etc.
