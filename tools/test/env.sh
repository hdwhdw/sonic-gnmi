#!/usr/bin/env bash

set -e

TOPDIR=$(realpath $(dirname ${BASH_SOURCE})/../..)

if [[ $# -gt 0 ]]; then
    . ${TOPDIR}/../sonic-mgmt-common/tools/test/env.sh "$@"
else
    . ${TOPDIR}/../sonic-mgmt-common/tools/test/env.sh \
        --dest=${TOPDIR}/build/test \
        --dbconfig-in=${TOPDIR}/testdata/database_config.json
fi
