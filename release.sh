#!/bin/bash
# example: sh scripts/release.sh --name 040620 --head svc-v0.15.0

IS_HOTPATCH=false

POSITIONAL=()
while [[ $# -gt 0 ]]
do
  key="$1"
  case $key in
      -n | --name)
        RELEASE_NAME="$2"
        shift # past argument
        shift # past value
        ;;
      -h | --head)
        RELEASE_HEAD="$2"
        shift # past argument
        shift # past value
        ;;
      -p | --patch)
        IS_HOTPATCH=true
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

if [[ $RELEASE_NAME == *"/"* ]]; then
    echo "release name should not contain '/' symbol"
    exit 1
fi

if [[ $IS_HOTPATCH == true && $RELEASE_HEAD == "" ]]; then
  echo "hotpatch requires the release head (tag or commit). use --head argument"
  exit 1
fi

if [[ $RELEASE_HEAD != "" ]]; then
    git fetch --tags
fi

BRANCH_NAME="release/$RELEASE_NAME"
if [[ $IS_HOTPATCH == true ]]; then
  BRANCH_NAME="hotpatch/$RELEASE_HEAD-$RELEASE_NAME"
fi

git checkout $RELEASE_HEAD -b $BRANCH_NAME
git push -u origin $BRANCH_NAME

echo "\n===> created and pushed release branch $BRANCH_NAME"