#!/bin/sh

case "$1" in
    "install")
        mkdir -p /usr/bin
        cp -rf ../cmd/supd/supd /usr/bin
        mkdir -p /etc/supd/conf.d
        cp -rf ../etc/supd/supd.ini /etc/supd
    ;;
"upgrade")
        cp -rf ../cmd/supd/supd /usr/bin
    ;;
    "clean")
        if [ -f "/usr/bin/supd" ];then
            sudo rm -rf /usr/bin/supd
        fi
    ;;
esac
