#!/bin/bash

set -ex

source ./cluster-up/hack/common.sh
source ./cluster-up/cluster/${KUBEVIRT_PROVIDER}/provider.sh

make undeploy || echo "this is fine"

port=$(./cluster-up/cli.sh ports registry | xargs)
# push to local registry provided by kvci
make docker-build IMG="127.0.0.1:${port}/kubevirt-migrations-controller:latest"
make docker-push IMG="127.0.0.1:${port}/kubevirt-migrations-controller:latest"
# the "cluster" (kvci VM) only understands the alias registry:5000 (which maps to 127.0.0.1:${port})
make deploy IMG="registry:5000/kubevirt-migrations-controller:latest"
