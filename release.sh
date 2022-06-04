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

if [[ $RELEASE_NAME == "" ]]; then
    echo "empty release name"
    exit 1
fi

BRANCH_NAME="release/$RELEASE_NAME"

if [[ $RELEASE_HEAD != "" ]]; then
    git fetch --tags
fi

git checkout $RELEASE_HEAD -b $BRANCH_NAME