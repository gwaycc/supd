#!/bin/sh

echo $PRJ_ROOT
./publish.sh
sudo PRJ_ROOT=$PRJ_ROOT ./supd
