#/bin/bash

bootName=""
case `uname` in
    "Linux"|"linux")
        sysname=$(cat /etc/issue|awk -F " " '{print $1}')
        case $sysname in
            "Debian")
                bootName="boot_debian.sh"
                ;;
            "Ubuntu")
                bootName="boot_debian.sh"
                ;;
            "CentOS")
                bootName="boot_centos.sh"
                ;;
            # TODO: more system support
        esac
    ;;
    "Darwin")
        bootName="boot_darwin.sh"
    ;;
    # TODO: more system support
esac

if [ -z "${bootName}" ]; then
    echo "System unknow"
    exit 0
fi

case "$1" in 
    "install")
        if [ ! -f "./cmd/supd/supd" ]; then
            echo "Program supd not found"
            exit 0
        fi
        cd script
        ./${bootName} install
        cd ..
        echo "Install done"
        ;;
    "upgrade")
        cd script
        ./${bootName} upgrade
        cd ..
        echo "Upgrade done"
        ;;
    "clean")
        cd script
        ./${bootName} clean
        cd ..
        echo "Clean done"
        ;;
    *)
        echo "install -- install to system."
        echo "upgrade -- reinstall supd binary and restart supd."
        echo "clean -- remove the installed."
        ;;
esac


