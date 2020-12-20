#!/bin/bash

srvHome=/Library/LaunchDaemons

case "$1" in 
    "install")
        if [ -f "$srvHome/supd.plist" ]; then
            echo "Already installed"
            exit 0
        fi
        sudo ./install_bin.sh install
        sudo cp supd.plist $srvHome/supd.plist
        sudo launchctl load $srvHome/supd.plist
        ;;
    "upgrade")
        if [ -f "$srvHome/supd.plist" ]; then
            sudo ./install_bin.sh upgrade
            echo "Upgrade the binary done. You need use 'launchctl' tools to restart supd by manually."

        else
            echo "Not installed"
        fi
        ;;
    "clean")
        if [ -f "$srvHome/supd.plist" ]; then
            sudo launchctl unload $srvHome/supd.plist
            sudo rm $srvHome/supd.plist
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
