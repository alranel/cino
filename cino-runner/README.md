# cino-runner

cino-runner runs tests on physical Arduino boards. It's a wrapper around the arduino-cli tool and can be used in manual mode or client mode.

## Installing cino-runner

Make sure you have go installed on your machine.

1. Clone the cino repo and cd to the cino-runner directory:

    ```
    git clone https://github.com/alranel/cino.git
    cd cino/cino-runner
    ```

2. Compile:

    ```
    go build
    ```

## Manual mode

Running cino-runner in manual mode is convenient when you want to run a test manually or you don't have a full cino infrastructure set up. To use it, you'll do just:

```
cino-runner run -b arduino:avr:uno -p /dev/ttyACM0 path/to/your/test
```

The value for `-b` should be a FBQN string representing a board.

This command will compile the test and upload it to the board connected to the given port, then it will connect to the serial port and parse the test results. If a test fails, cino-runner exists with a non-zero value.

If the direct path to a cino test (i.e. a directory containing a *cino.yml* file) is supplied, the test will be run. If a *cino.yml* file is not found in the path, its subdirectories are traversed recursively in order to find all the runnable tests. This allows you to just supply the path to a repository containing tests.

If the path points to a directory containing an **Arduino library** (detected by the presence of a *library.properties* file), that library is included in the compilation. Likewise, if the path points to a directory containing an **Arduino core** (detected by the presence of a *boards.txt* file), that core is installed before running the test. Of course it will be actually used only if the board FQBN refers to it.

### Using a configuration file

To avoid passing board FBQN and port every time, you can put everything in a configuration file and invoke the tool like this:

```
cino-runner run -c config.yml path/to/test1
```

To create a configuration file, just copy the sample one:

```
cp config.yml.example config.yml
vi config.yml
```

The following configuration options are available:

* **devices**: the list of physical devices connected to your instance. For each one, the following keys can be configured:
  * **fqbn**: (Required) The FQBN describing the board type, such as arduino:avr:uno. Use `arduino-cli board list` to see the FQBN of the connected boards, or `arduino-cli board listall` to see the full list.
  * **port**: (Required) The path to the device, such as /dev/cu.usbmodem14101. Make sure the assigned path [does not change](https://unix.stackexchange.com/questions/66901/how-to-bind-usb-device-under-a-static-name) across restarts or device resets.
  * **features**: A list of free tags representing features of the board, such as `wifinina` or `ble5`. This is used to check if the device satisfies the requirements expressed in test metadata.

## Client mode

In client mode, the tool subscribes to a cino-server and waits for an available job matching the local capabilities. Matching tests are executed and results are posted back to cino-server.

```
cino-runner subscribe -c config.yml
```

When running in client mode, the following options are also required in the configuration file:

* **runner_id**: the ID of this runner instance. It can be any string, as long as it's unique in your cino pool. Make sure it is listed in the cino-server configuration too.
* **db.dsn**: the credentials to use when accessing the PostgreSQL server. Make sure they match the ones configured in cino-server/.env

### Running in daemon mode

You can choose your favorite way to run cino-runner as a daemon, including systemd or launchctl.

A sample Dockerfile is also provided, but keep in mind that `docker run` will need the `--device` option to access the devices, and that does not currently work on macOS. The Docker image can be compiled like this:

```
docker build -t cino-runner -f Dockerfile ..
```
