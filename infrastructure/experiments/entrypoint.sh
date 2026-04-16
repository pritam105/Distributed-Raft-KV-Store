#!/bin/bash
# SERVICE_TYPE controls which binary runs: "node" (default) or "simplekvs"
if [ "${SERVICE_TYPE:-node}" = "simplekvs" ]; then
    exec simplekvs
else
    exec node
fi
