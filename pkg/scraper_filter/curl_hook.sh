#!/bin/bash

PROG_PATH=$(readlink -f $0)
PROG_DIR=${PROG_PATH%/*}

CURL_HOOK_DIR="${PROG_DIR}/../../internal/curl_hook"
ORDERING_DIR="${PROG_DIR}"

export FILTER_PLUGIN="scraper_filter.so"
export ORDERING_CONFIG="${ORDERING_DIR}/ordering.yaml"
export LD_LIBRARY_PATH=${CURL_HOOK_DIR}
export LD_PRELOAD=$(readlink -f ${CURL_HOOK_DIR}/curl_hook.so)
curl "$@"
