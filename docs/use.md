Adding catalog items manually
-----------------------------

    $ cf brooklyn add-catalog [<broker> <username> <password>] <path/to/bluepri$

this allows new entities to be created and added to the brooklyn
catalog.  The service broker that is associated will need to be
refreshed with `cf update-service-broker` and enabled with
`enable-service-access` for these new services to become available.

Deleting catalog items
----------------------

    $ cf brooklyn delete-catalog [<broker> <username> <password>] <name> <versi$

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
