# Set up environment so that every process is injected with curl_hook.so

if [ $# -ne 1 ]; then
	echo "missing FILTER_PLUGIN argument"
	exit
fi

PROG_PATH=$(readlink -f $BASH_SOURCE)
PROG_DIR=${PROG_PATH%/*}

export LD_LIBRARY_PATH=${PROG_DIR}
export LD_PRELOAD=curl_hook.so
export FILTER_PLUGIN="$(readlink -f $1)"
