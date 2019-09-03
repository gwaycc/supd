#!/bin/bash

# Template from https://github.com/gwaycc/goapp

# 项目需要导出的程序环境变量
# -------------------------------------------------
export PRJ_ROOT=`pwd`
export PRJ_NAME="supd"
export GOBIN=$PRJ_ROOT/bin
export GO111MODULE=on

# 设定sup [command] all 所遍历的目录
export BUILD_ALL_PATH="cmd/supd"

# supervisord配置文件参数
## --------------------START-------------------
## 以下是部署时的supervisor默认配置数据，若未配置时，会使用以下默认数据
## 开发IDE可不配置以下环境变量
## 配置supervisor运行的用户，默认为当前用户
#export SUP_USER=$USER
## 配置supervisor的配置文件目录
#export SUP_ETC_DIR="/etc/supervisor/conf.d/" # (可选)
## 配置supervisor的子程序日志的单个文件最大大小
#export SUP_LOG_SIZE="10MB"
## 配置supervisor的子程序日志的最多文件个数
#export SUP_LOG_BAK="10"
## 配置supervisor配置中的environment环境变量
#export SUP_APP_ENV="PRJ_ROOT=\\\"$PRJ_ROOT\\\",GIN_MODE=\\\"release\\\",LD_LIBRARY_PATH=\\\"$LD_LIBRARY_PATH\\\""

# 设定publish指令打包时需要包含的文件夹环境变量
# -------------------------------------------------
# 默认会打包以下目录：$PRJ_ROOT/bin/* $BUILD_ALL_PATH等二进制程序
export PUB_ROOT_RES="etc script setup.sh" # 根目录下需要打包的文件夹列表，如"etc"等, 空格字符串表示不使用
# export PUB_APP_RES="public" # app下的文件夹列表，如"res public"等, 空格字符串表示不使用

# 更改路径可更改编译器的版本号, 如果未指定，使用系统默认的配置
go_root="/usr/local/go"
if [ -d "$go_root" ]; then
    export GOROOT="$go_root"
fi

# 将GOBIN加入PATH
bin_path=$GOBIN:$GOROOT/bin:

rep=${PATH/bin_path/""}
if [ ! "$PATH" = "$rep" ]; then
    PATH=$rep # 重新设定原值的位置
fi
export PATH=$bin_path$PATH

# 下载sup管理工具
if [ ! -f $GOBIN/sup ]; then
    type curl >/dev/null 2>&1||{ echo -e >&2 "curl not found, need install at first."; exit 0; }
    echo "Download sup to bin."
    mkdir -p $GOBIN&& \
    curl https://raw.githubusercontent.com/gwaycc/supd/v1/bin/sup -o $GOBIN/sup && \
    chmod +x $GOBIN/sup&&echo "Download sup done."|| exit 0
fi

# 设定git库地址转换, 以便解决私有库中https证书不可信的问题
# git config --global url."git@git.gway.cc:".insteadOf "https://git.gway.cc"
# --------------------END--------------------

echo "Env have changed to \"$PRJ_NAME\""
echo "Using \"sup help\" to manage project"
# -------------------------------------------------

