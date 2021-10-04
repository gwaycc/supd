#/bin/bash

bootName=""
case `uname` in
    "Linux"|"linux")
        files="/etc/redhat-release /etc/issue"
        for file in $files
        do
           if [ ! -f "$file" ]; then
               continue
           fi

           sysname=$(cat $file |awk -F " " '{print $1}')
           case $sysname in
               "Debian")
                   bootName="boot_debian.sh "
                   break
                   ;;
               "Ubuntu")
                   bootName="boot_debian.sh "
                   break
                   ;;
               "CentOS")
                   bootName="boot_centos.sh "
                   break
                   ;;
           	# TODO: more system support
           esac
        done
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
        if [ ! -f "./supd" ]; then
            echo "Program supd not found, need build first."
            exit 0
        fi
        cd script
        ./${bootName} install
        cd ..
        echo "Install supd done"
        ;;
    "upgrade")
        if [ ! -f "./supd" ]; then
            echo "Program supd not found, need build first."
            exit 0
        fi
        cd script
        ./${bootName} upgrade
        cd ..
        echo "Upgrade supd done"
        ;;
    "clean")
        cd script
        ./${bootName} clean
        cd ..
        echo "Clean supd done"
        ;;
    *)
        echo "install -- install to system."
        echo "upgrade -- reinstall supd binary and restart supd."
        echo "clean -- remove the installed."
        ;;
esac


