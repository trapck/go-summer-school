#!/bin/bash
# required go install github.com/maykonlf/semver-cli/cmd/semver@v1.1.0
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

echo "$RELEASE_NAME ; $RELEASE_HEAD"

BRANCH_NAME="release/$RELEASE_NAME"

git checkout $RELEASE_HEAD -b $BRANCH_NAME