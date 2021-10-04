#!/bin/sh

go build -ldflags="-s -w"||exit 0

echo "Publish done"
