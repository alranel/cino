# 🐦 cino

_Continuous Integration for Arduino_

cino provides on-hardware continuous integration for GitHub repositories containing Arduino cores, libraries and sketches. It's fully open source and it's based on the powerful [arduino-cli](https://github.com/arduino/arduino-cli) tool.

> ⚠️ Warning: this is very experimental.

## Overview

### Anatomy of a test

A test is a normal Arduino sketch containing one or more assertions implemented using the `REQUIRE()` and `CATCH()` macros borrowed by Catch2 syntax, preceded by a `TEST_PLAN()` macro used to declare the expected number of tests. See the [examples](examples/) directory.

```c++
#include <cino.h>

void setup() {
    TEST_PLAN(1);
    
    Wire.setClock(100000);
    REQUIRE( TWI0.MBAUD == 67 );
}
```

As an alternative to `TEST_PLAN()` (recommended), a combination of `TEST_NO_PLAN()` and `TEST_DONE()` can be used in case it is not possible to determine the number of tests in advance.

A `cino.yml` file is required within the sketch folder to signal that the sketch is a test. The file can be empty, but can be used to specify any hardware requirements used to route the test to a suitable runner:

```yaml
sketches:
  - require-features: 
    - wifinina
```

Requirements are just free tags: tests will be run as long as there's a configured instance of cino-runner having the correspondent tags, or skipped if there are none. Multiple requirements can be specified for a single test, which means that it will be run on a board that satisfies all of them.

The resulting directory structure looks like this:

```
01_signals
├── 01_signals.ino
└── cino.yml
```

### Multi-board tests

There might be situations where a test involves multiple boards, connected one to each other. This is needed for instance when testing communication protocols or any hardware behavior that can be checked with an external probe. In this case, the cino.yml file would include multiple entries under the `sketches` key, each one with a subdirectory name:

```yaml
require-wiring:
  - i2c
sketches:
  - dir: main
    require-features:
      - wifinina
  - dir: probe
```

Note that we have a set of hardware requirements for each sketch, and then an additional set of requirements at a global level which can be used to represent the hardware configuration (like wiring or any additional components).

Important: cino will try to run a multi-board test according to all the available board combinations, unless you specify some restrictions. Supposing you're running the above example on a runner having two devices, both supporting the `wifinina` feature: cino will run the test **twice**, inverting the sketches assigned to each board. When this is not desired you need to add more restrictions using the `require-features`, `require-architecture`, `require-fqbn` keys:

```yaml
sketches:
  - dir: main
    require-features:
      - wifinina
  - dir: probe
    require-fqbn: arduino:samd:nano_33_iot
```

> Hint: if a sketch can be run on any board (which is often the case for the sketches used as probes) and you don't care about repeating it on multiple devices, just set `require-fqbn` or `require-architecture` to `*`. This will assign the first available board.

When a test involves multiple sketches, the directory structure looks like this:

```
03_I2C
├── cino.yml
├── main
│   └── main.ino
└── probe
    └── probe.ino
```

cino does not provide a synchronization mechanism between multiple boards involved in a single test: if needed, it can be done with some wiring and implementing the related logic directly in the sketches.

### Testing a core or a library

Tests can be put in any directory within a repository. For a core or a library, it could be a good idea to put everything under a `hwtest` directory located in the root of the repository:

```
hwtest
├── 01_signals
│   ├── 01_signals.ino
│   └── cino.yml
└── 02_timing
    ├── 02_timing.ino
    └── cino.yml
```

When testing a repository containing a core or a library, cino automatically runs the tests against the local version of that core or library.

> TODO: this is currently only implemented for libraries.

### Testing a sketch

You have two options here. You could add your tests under their own directory, like explained above for cores and libraries; however you could just do everything within your main sketch. Just include `cino.h` and put the `REQUIRE()` and `CHECK()` assertions within your code. When compiling from the Arduino IDE or arduino-cli, they will be ignored. When run under a cino-runner instance, they will be executed.

> TODO: this is not available yet as it requires the [cino library](cino-library) to be indexed by the Arduino Library Manager and a `-D` flag to be [supported by arduino-cli](https://github.com/arduino/arduino-cli/issues/159).

### Architecture

A typical setup has:

* an instance of [cino-server](cino-server) installed on a publicly reachable server, providing:
  * a REST API to receive notifications from GitHub
  * a job queue based on PostgreSQL
* a pool of instances of [cino-runner](cino-runner) installed on physical machines with one or more Arduino boards attached (also known as *pool-cino*)

Attaching multiple boards to a single cino-runner instance allows to run multi-board tests, as long as their hardware capabilities match the available boards. On the other hand, attaching each board to its own cino-runner instance allows for faster runs of single-board tests because they get parallelized even on the same machine.

### Flow

When someone submits a Pull Request, the following happens:

1. GitHub calls a webhook exposed by cino-server notifying the repository and the reference of the commit to test.
2. cino-server clones the repo and looks for tests to run.
3. For each unique set of requirements, a job is inserted in a queue along so that each runner can pick up the ones they are compatible with.
  * When testing a library, such jobs are replicated for each architecture that the library is compatible with.
  * When testing a core, such jobs replicated for each board FQBN supported by the core.
4. Instances of cino-runner subscribe to the jobs queue and retrieve the pending jobs. If they can't handle a job, they mark it in the queue.
5. For each job, cino-runner clones the repository and runs all the available tests uploading the results to the job queue.
6. Each test gets compiled with arduino-cli, uploaded to the board(s) and run. Serial output is captured by cino-runner and parsed.
7. cino-server watches the job status and calls the GitHub API to notify the test results whenever a job status changes (in progress, success, failure) or whenever a job was skipped by all the runners (in this case it is marked as skipped in GitHub).

## Security considerations

Untrusted users can open a pull request that includes malicious code within the test or in the main codebase, that will be run on hardware. This can be harmful in the following circumstances:

* When the board has direct access to external resources
    * Boards are supposed to run in isolated environments but this is hard to do when it comes to wireless/radio connectivity: an attacker could scan wifi networks or perform radio communications.
      * As a mitigation, no serial output is included in the visible output so there's no way for an attacker to read captured data.
* When the code can do destructive actions on the board
    * For instance, replacing firmware on other board components.
      * What mitigation is doable for this?
    * ...?

## Getting started

See the README files for [cino-server](cino-server) and [cino-runner](cino-runner) for guidance about installing cino. If you want to play around, you can just start with executing cino-runner on local tests (there are a few in the [examples](examples) directory) which does not need a running server.

## License

cino-server and cino-runner are provided under the terms of the AGPL-3.0 license.

The cino library is provided under the terms of the MIT library for easier inclusion in any other project.

cino was developed with 💙 by Alessandro Ranellucci.
