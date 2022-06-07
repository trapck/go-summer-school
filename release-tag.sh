#!/bin/bash

# Script helps to create git tags for releases/hotpatches.
# Requires semver utility to be installed (go install github.com/maykonlf/semver-cli/cmd/semver@v1.1.0)

# The service argument (--service or -s) is the only required argument and should be
# a valid service name from the /services folder with the .semver.yaml file.
# A tag is created based on the service name and its version from the .semver.yaml.

# By default the tag is created and pushed to the remote. You can change this behaviour by
# providing the (--skip-push) flag. It is not recommended to use --skip-push flag
# for normal flows. Use it only in special cases and for debug/test scenarios.


# Example:
# sh scripts/release-tag.sh --service myservice [--skip-push]

IS_SKIP_PUSH=false

POSITIONAL=()
while [[ $# -gt 0 ]]
do
    key="$1"
    case $key in
        -s | --service)
            SERVICE_NAME="$2"
            shift # past argument
            shift # past value
            ;;
        --skip-push)
            IS_SKIP_PUSH=true
            shift
            shift
            ;;
    esac
done
set -- "${POSITIONAL[@]}"

if [[ $SERVICE_NAME == "" ]]; then
    echo "empty service name"
    exit 1
fi

SERVICE_PATH="./services/$SERVICE_NAME"

cd "services/$SERVICE_NAME"

VERSION=$(semver get release)
          
if [[ $(semver get alpha) != *-alpha.0 ]]; then
  VERSION=$(semver get alpha)
elif [[ $(semver get beta) != *-beta.0 ]]; then
  VERSION=$(semver get beta)
elif [[ $(semver get rc) != *-rc.0 ]]; then
  VERSION=$(semver get rc)
fi

GIT_TAG="$SERVICE_NAME-$VERSION"

git tag -a $GIT_TAG -m "$SERVICE_NAME svc tag $GIT_TAG"
echo "\n===> created tag $GIT_TAG in local reposiotry\n"

if [[ $IS_SKIP_PUSH == false ]]; then
    git push origin $GIT_TAG
    echo "\n===> pushed tag $GIT_TAG to the remote repository"
fi