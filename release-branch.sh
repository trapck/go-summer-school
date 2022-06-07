#!/bin/bash

# Script helps to create release/hotpatch branches.

# The name argument (--name or -n) value is the only required argument and will appear 
# after the release/ or hotpatch/ prefix in the created branche's name.
# The value should be a valid git branch name and should not include '/' characters.

# By default release/ branch is created. If you want hotpatch branch, provide the (--patch or -p) flag.

# By default the branch is created from the head of the current branch. If you want to create branch
# from some certain commit of tag, you can provide it by the (--head or -h) argument. Script fetches all
# the remote tags before creating the branch so it is not required to have the pre-fetched local tag.
# But if you want to create the branch from some of the previous commits make sure you have that commit
# in your local repository (make your branch up to date with the remote).
# Note that it is allowed to create hotpatch branhces only with provided --head argument. 

# By default the branch is created and pushed to the remote. You can change this behaviour by
# providing the (--skip-push) flag. It is not recommended to use --skip-push flag
# for normal flows. Use it only in special cases and for debug/test scenarios.

# Example:
# sh scripts/release-branch.sh --name 040620 [--head svc-v0.15.0] [--patch] [--skip-push]
# Take care to add extra -- if bool flags are passed one by one (e.g. --patch -- --skip-push)

IS_HOTPATCH=false
IS_SKIP_PUSH=false

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
        shift
        shift
        ;;
      -p | --patch)
        IS_HOTPATCH=true
        shift
        shift
        ;;
      --skip-push)
        IS_SKIP_PUSH=true
        shift
        shift
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
echo "\n===> created and switched to $BRANCH_NAME\n"

if [[ $IS_SKIP_PUSH == false ]]; then
  git push -u origin $BRANCH_NAME
  echo "\n===> pushed $BRANCH_NAME to the remote repository"
fi