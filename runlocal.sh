#!/bin/sh

go build ./cmd/blockchainminiproject

MAX_TRANSACTIONS=3 TIME_LIMIT=1 ./blockchainminiproject