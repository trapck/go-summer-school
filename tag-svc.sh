#!/bin/bash
# required go install github.com/maykonlf/semver-cli/cmd/semver@v1.1.0
# example: sh scripts/tag-svc.sh --service myservice

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    --service)
    SERVICE_NAME="$2"
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

#git tag -a $GIT_TAG -m "$SERVICE_NAME svc tag $GIT_TAG"

echo "===> created tag $GIT_TAG in local reposiotry\n"

#git push origin $GIT_TAG

echo "\n===> pushed tag $GIT_TAG to remote repository"