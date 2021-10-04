#!/bin/sh

case "$1" in
    "install")
        mkdir -p /usr/bin
        cp -rf ../supd /usr/bin
        cp -rf ../supc /usr/bin
        mkdir -p /etc/supd/conf.d
        cp -rf ../etc/supd/supd.ini /etc/supd
    ;;
    "upgrade")
        cp -rf ../supd /usr/bin
        cp -rf ../supc /usr/bin
    ;;
    "clean")
        if [ -f "/usr/bin/supd" ];then
            rm /usr/bin/supd
            rm /usr/bin/supc
        fi
    ;;
esac
