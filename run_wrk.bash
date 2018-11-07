#!/usr/bin/env bash

curl http://localhost:8080/startProfiling
wrk "$@"
curl http://localhost:8080/stopProfiling
