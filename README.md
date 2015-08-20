Parse Command Line Tool
=======
[![Build Status](https://travis-ci.org/ParsePlatform/parse-cli.svg?branch=master)](https://travis-ci.org/ParsePlatform/parse-cli)

The `Parse Command Line Tool` allows you to interact with your Cloud Code from the terminal.

For more information please visit [Parse](https://parse.com/docs/js/guide#command-line).

Overview
--------
Parse Command Line Tool can be used to perform various actions on your Parse app.
It can be used to create new Parse apps, deploy Cloud Code to an app, view all releases for an app, etc.

You can install `Parse Command Line Tool` by following instructions at [parse-cli](https://parse.com/apps/quickstart#cloud_code/).

Alternatively, you can just type the following command.

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
  functions   List Cloud Code functions and function webhooks
  generate    Generates a sample express app in the current project directory
  jssdk       Sets the Parse JavaScript SDK version to use in Cloud Code
  list        Lists Parse apps associated with current Parse account
  logs        Prints out recent log messages
  new         Creates a new Parse app and adds Cloud Code to an existing Parse app
  releases    Gets the releases for a Parse app
  rollback    Rolls back the version for the given app
  symbols     Uploads symbol files
  triggers    Lists Cloud Code triggers and trigger webhooks
  update      Updates this tool to the latest version
  version     Gets the Command Line Tools version
  help        Help about any command
```
