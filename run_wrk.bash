#!/usr/bin/env bash

wrk "$@"
curl http://localhost:8080/stop
