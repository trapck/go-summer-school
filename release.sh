#!/bin/bash
# example: sh scripts/release.sh --name 040620 --head svc-v0.15.0

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --name)
    RELEASE_NAME="$2"
    shift # past argument
    shift # past value
    ;;
    --head)
    RELEASE_HEAD="$2"
    shift # past argument
    shift # past value
    ;;
esac
done
set -- "${POSITIONAL[@]}"

if [[ $SERVICE_NAME == "" ]]; then
    echo "empty service name"
    exit 1
fi

BRANCH_NAME="release/$RELEASE_NAME"

git checkout $RELEASE_HEAD -b $BRANCH_NAME