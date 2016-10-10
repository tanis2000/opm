[![POGODEV](https://github.com/pogodevorg/assets/blob/master/public/img/logo-github.png?raw=true)](https://pogodev.org)

# OpenPokeMap
## Table of Contents

* [What is it?](#what-is-it)
* [Documentation](#documentation)
  * [Installation](#installation)
  * [Requirements](#requirements)
  * [Configuration](#configuration)
* [Licensing](#licensing)
* [Contributing](#contributing)
  * [Core Maintainers](#core-maintainers)
* [Credits](#credits)

## What is it?
`opm` contains the complete OPM stack.
- `/apiserver` - http endpoint for all OPM api calls
- `/bancheck` - service that checks if accounts flagged as banned are really banned
- `/buildscripts` - build/install scripts for windows and linux
- `/db` - package for interfacing with the OPM database (MongoDB)
- `/opm` - OPM specific stuff
- `/proxyhub` - Proxy layer for OPM infrastructure
- `/scanner` - Performs actual scans for monsters and stuff
- `/stats` - Provides common statistics
- `/tools` - Random tools for testing/development
- `/util` - Utility stuff 

## Documentation
### Requirements
- Go - [https://golang.org/]()

### Installation
1. Run `go get github.com/pogointel/opm`
3. Run `buildscripts/[windows|linux]/install.[bat|sh]` (choose the right one for your platform)

### Configuration
Soon.

## Licensing
[GNU GPL v3](https://github.com/pogointel/opm/blob/master/LICENSE)

## Contributing
Currently, you can contribute to this project by:
* Joining us on [Discord](https://discord.pogodev.org/) in the `#pogointel` channel.
* Submitting a detailed [issue](https://github.com/pogointel/opm/issues/new).
* [Forking the project](https://github.com/pogointel/opm/fork), and sending a pull request back to for review.

### Core Maintainers

* [BadLamb](https://github.com/BadLamb)
* [nullpixel](https://github.com/nullpixel1)
* [femot](https://github.com/femot)
* [joelfi](https://github.com/joelfi)

## Credits
* [Lisiano](https://github.com/Lisiano256) :heart: - public relations and devops 
