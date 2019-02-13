#!/bin/bash

PROG_PATH=$(readlink -f $0)
PROG_DIR=${PROG_PATH%/*}

export LD_PRELOAD=$(readlink -f ${PROG_DIR}/curl_hook.so)
curl "$@"
