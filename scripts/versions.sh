#!/usr/bin/env bash

# Set up the versions to be used
coreth_version=${CORETH_VERSION:-'v0.7.1'}
# Don't export them as they're used in the context of other calls
avalanche_version=${AVALANCHE_VERSION:-'v1.6.2-rc.2'}