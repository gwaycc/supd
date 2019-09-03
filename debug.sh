#!/bin/sh

echo $PRJ_ROOT
sup build all
sudo PRJ_ROOT=$PRJ_ROOT ./cmd/supd/supd
