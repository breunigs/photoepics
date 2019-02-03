#!/usr/bin/env bash

trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT

target="$(dirname $0)/installed/"
if [ ! -x "${target}/dgraph" ] || [ ! -x "${target}/dgraph-ratel" ]; then
  /usr/bin/env bash $(dirname $0)/get.sh
fi

cd "${target}"

./dgraph zero&
./dgraph alpha --lru_mb 1024 --zero localhost:5080&
./dgraph-ratel&

wait
