#!/bin/bash

srvHome=/lib/systemd/system

case "$1" in 
    "install")
        if [ -f "$srvHome/supd.service" ]; then
            echo "Already installed"
            exit 0
        fi
        sudo ./install_bin.sh install
        sudo cp supd.service $srvHome/supd.service 
        sudo systemctl daemon-reload
        sudo systemctl enable supd
        sudo systemctl start supd
        ;;
    "upgrade")
        if [ -f "$srvHome/supd.service" ]; then
            sudo ./install_bin.sh upgrade
            sudo systemctl restart supd
        else
            echo "Not installed"
        fi
        ;;
    "clean")
        if [ -f "$srvHome/supd.service" ]; then
            sudo systemctl stop supd
            sudo systemctl disable supd
            sudo rm $srvHome/supd.service 
            sudo systemctl daemon-reload
            sudo ./install_bin.sh clean
        else
            echo "Not installed"
        fi
        ;;
    *)
        echo "install -- to install on system boot"
        echo "clean -- to remove system bootable"
        ;;
esac
