PixelGW, a self-hosted Pixlet server for low-resolution devices.

Maintainer: Joe Sunday sunday@csh.rit.edu
GitHub: https://github.com/joe714/pixelgw

# Operation
PixelGW runs [Pixlet](https://github.com/tidbyt/pixlet) apps and serves them
to one or more devices over Websockets. See https://github.com/joe714/pixelclient
for the device firmware.

The server configures one or more channels, which is a list of apps and
subscribed devices. By default, new devices will register against the
*default* channel which comes preconfigured; more channels can be created
and configured and devices can change what they are subscribed to.

Devices connect to the server at ws://*ip:port*/ws?device=*deviceUUID*, and
new webp images are streamed to the device as the applets are executed.
Example device firmware is coming soon.

Currently the server is intended for single tenant use on a secured
home network, and there is no user validation for the REST APIs.

# Compile and deploy
PixelGW is primarily built and run as a docker image.
You will need Docker installed and configured with your user in the docker group.

After initially cloning this repository, get the submodules:

    $ git submodule init && git submodule update

To rebuild the generated REST bindings after editing pixelgw.yaml:

    $ make generate

To build the docker image:

    $ make

There are two default deploy targets: *deploy_prod* will listen on port 8080,
*deploy_test* will listen on 8081:

    $ make deploy_prod

# Applets
Applets are built into the docker image from the contents of the /apps directories:
- /apps/community - Sync from the Tidbyt community depot of third party apps
- /apps/local - Local apps for specific installs go here.

# API

The REST API is under heavy development and subject to breaking changes
at this point. The full API documentation is in pixelgw.yaml, and allows
creating new channels and configuring applets and device subscriptions.

Full examples to come.

# Limitations
Some Pixlet features are not yet supported:
- OAuth parameters
- Audio (Tidbyt2)

See the TODO.md for the full roadmap.

