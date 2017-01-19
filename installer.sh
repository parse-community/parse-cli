#!/bin/bash

TMP_FILE=/tmp/back4app.tmp
if [ -e ${TMP_FILE} ]; then
  echo "Cleaning up from previous install failure"
  rm -rf ${TMP_FILE}
fi
echo "Fetching latest version ..."

export latest=3.0.6-beta-2

case `uname` in
  "Linux" )
    #export latest=`curl -X GET https://api.parse.com/1/supported?version=latest|grep -Po '(\d.\d.\d)'`
    export url="https://github.com/back4app/parse-cli/releases/download/release_${latest}/b4a_linux"
   ;;
  "Darwin" )
    #export latest=`curl -X GET https://api.parse.com/1/supported?version=latest|grep -Eo '(\d.\d.\d)'`
    export url="https://github.com/back4app/parse-cli/releases/download/release_${latest}/b4a"
    ;;
esac

curl --progress-bar --compressed -Lo ${TMP_FILE} ${url}

if [ ! -d /usr/local/bin ]; then
  echo "Making /usr/local/bin"
  mkdir -p /usr/local/bin
fi
echo "Installing ..."
mv /tmp/back4app.tmp /usr/local/bin/b4a
chmod 755 /usr/local/bin/b4a
