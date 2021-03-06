# trusty

[![Build Status](https://travis-ci.com/go-phorce/trusty.svg?branch=master)](https://travis-ci.com/go-phorce/trusty)
[![Coverage Status](https://coveralls.io/repos/github/go-phorce/trusty/badge.svg?branch=master)](https://coveralls.io/github/go-phorce/trusty?branch=master)

Trusty is a Certification Authority.

## Requirements

1. GoLang 1.14+
1. SoftHSM 2.5+

## Build

* `make all` initializes all dependencies, builds and tests.
* `make proto` generates gRPC protobuf.
* `make build` build the executable
* `make gen_test_certs` generate test certificates
* `make test` run the tests
* `make testshort` runs the tests skipping the end-to-end tests and the code coverage reporting
* `make covtest` runs the tests with end-to-end and the code coverage reporting
* `make coverage` view the code coverage results from the last make test run.
* `make generate` runs go generate to update any code gen'd files (query_console.go in our case)
* `make fmt` runs go fmt on the project.
* `make lint` runs the go linter on the project.

run `make all` once, then run `make build` or `make test` as needed.

First run:

    make all

Subsequent builds:

    make build

Tests:

    make test

Optionally run golang race detector with test targets by setting RACE flag:

    make test RACE=true

Review coverage report:

    make coverage

## Generate protobuf

    make proto

## Local testing

When runnning unit tests, the Unix sockets are used. 
If the test fails, there can be `localhost:{port}` files left on the disk.
To clean up use:
    
    find -name "localhost:*" -delete

