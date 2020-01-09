#!/bin/sh

go build -o ./runblockchainminiproject ./cmd/blockchainminiproject

MAX_TRANSACTIONS=3 TIME_LIMIT=1 ./runblockchainminiproject