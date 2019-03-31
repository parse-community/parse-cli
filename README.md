Parse Command Line Tool
=======
[![Build Status](https://travis-ci.org/parse-community/parse-cli.svg?branch=master)](https://travis-ci.org/parse-community/parse-cli)
[![Join The Conversation](https://img.shields.io/discourse/https/community.parseplatform.org/topics.svg)](https://community.parseplatform.org/c/parse-server)
[![Backers on Open Collective](https://opencollective.com/parse-server/backers/badge.svg)](#backers)
[![Sponsors on Open Collective](https://opencollective.com/parse-server/sponsors/badge.svg)](#sponsors)
[![License][license-svg]][license-link]
![Twitter Follow](https://img.shields.io/twitter/follow/ParsePlatform.svg?label=Follow%20us%20on%20Twitter&style=social)

The `Parse Command Line Tool` allows you to set up your Parse app's server-side code from the terminal.
You can deploy your server-side code to either Parse Cloud Code or Heroku Node.js.

Overview
--------
Parse Command Line Tool can be used to perform various actions on your Parse app.
It can be used to create new Parse apps, deploy Cloud Code to an app, view all releases for an app, etc.

To install `Parse Command Line Tool`, you can just type the following command.

NOTE: You should already have [Go](https://golang.org/doc/install) installed and GOPATH, GOROOT set to appropriate values.

```bash
  go get -t github.com/ParsePlatform/parse-cli
```

This installs a binary called `parse-cli` at `$GOPATH/bin`.
Further, you can find the code for Parse CLI at: `$GOPATH/src/github.com/ParsePlatform/parse-cli`.

The following commands are currently available in the `Parse Command Line Tool`:
```bash
  add         Adds a new Parse app to config in current Cloud Code directory
  configure   Configure various Parse settings
  default     Sets or gets the default Parse app
  deploy      Deploys a Parse app
  develop     Monitors for changes to code and deploys, also tails parse logs
  download    Downloads the Cloud Code project
  functions   List Cloud Code functions and function webhooks
  generate    Generates a sample express app in the current project directory
  jssdk       Sets the Parse JavaScript SDK version to use in Cloud Code
  list        Lists properties of the given Parse app and Parse apps associated with given project
  logs        Prints out recent log messages
  migrate     Migrate project config format to the preferred format
  new         Adds Cloud Code to an existing Parse app, can also create a new Parse app
  releases    Gets the releases for a Parse app
  rollback    Rolls back the version for the given app
  symbols     Uploads symbol files
  triggers    List Cloud Code triggers and trigger webhooks
  update      Updates this tool to the latest version
  version     Gets the Command Line Tools version
  help        Help about any command
```

-----

As of April 5, 2017, Parse, LLC has transferred this code to the parse-community organization, and will no longer be contributing to or distributing this code.

[license-svg]: https://img.shields.io/badge/license-BSD-lightgrey.svg
[license-link]: LICENSE

