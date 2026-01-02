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

# PixelClient

PixelClient is a terminal UI client included with PixelGW that simulates an LED matrix
display in your terminal. It connects to the server via WebSocket and renders received
WebP frames as colored ASCII art.

## Building

    $ go build -o bin/ cmd/pixelclient/main.go

Or it will be built automatically as part of `make build`.

## Usage

```
pixelclient [flags]

Flags:
  -c, --config <path>      Path to config file (default: ~/.pixelclient/config.json)
  -d, --device <name>      Use named device from config (bypass menu)
  -n, --new                Create new device (interactive if missing args)
      --server <ip:port>   Server address (used with --new)
      --device-id <uuid>   Custom device UUID (optional, used with --new)
      --headless           Run without TUI, print frame info to stdout
      --time <seconds>     Auto-disconnect after N seconds (with --headless)
```

## Examples

```bash
# First run - create a new device interactively
pixelclient --new
# Prompts for: Device name, Server (ip:port)

# Create a device with all parameters
pixelclient --new -d living-room --server 192.168.1.100:8080

# Connect using a saved device
pixelclient -d living-room

# Run without arguments to auto-select "default" device or show menu
pixelclient

# Headless mode for testing (prints frame info without TUI)
pixelclient --headless -d living-room

# Headless with auto-disconnect after 30 seconds
pixelclient --headless --time 30 -d living-room
```

## Configuration

Device configurations are stored in `~/.pixelclient/config.json`. Each device
entry contains a name, server address, and UUID. The client will auto-select
a device named "default" if one exists, otherwise it shows a selection menu.

## TUI Controls

- **q** or **Ctrl+C**: Quit
- **↑/↓**: Navigate device list (when in menu)
- **Enter**: Select device

# Compile and deploy
PixelGW is primarily built and run as a docker image.
You will need Docker installed and configured with your user in the docker group.

After initially cloning this repository, get the submodules:

    $ git submodule init && git submodule update

To rebuild the generated REST bindings after editing pixelgw.yaml:

    $ make generate

To build the docker image:

    $ make

There are two default deploy targets:
- *make deploy_prod* will deploy the single complete production image on port 8080.
- *make deploy_test* will deploy the server image as well as live node image running
  the web UI from the source tree with HMR enabled on port 8081.

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

