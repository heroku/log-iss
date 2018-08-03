#!/usr/bin/env bash

function ecr_find_or_create {
  aws ecr create-repository --repository-name $1 >/dev/null 2>&1 || true
  echo "`aws ecr describe-repositories --repository-name $1 | jq '.repositories[0].repositoryUri' | sed -e 's/"//g'`"
}

function ecr_url {
  echo "`aws ecr describe-repositories | jq '.repositories[0].repositoryUri' | sed -e 's/"//g' | awk -F '/' '{ print $1 }'`"
}

function ecr_push {
  # Push the repo secified as the second argument.
  # push
  local image="$1"
  local version="$2"
  local ecr="`ecr_find_or_create ${image}`"

  set -uex
  docker tag $image:$version $ecr:$version
  docker tag $image:$version $ecr:latest

  eval `aws ecr get-login --no-include-email`
  docker push $ecr:$version
  docker push $ecr:latest
}

if test -z $1; then
  # Fetch only the repo URL. There must be at least one for this to work.
  ecr_url
elif [ "$1" = "push" ]; then
  # Push the repo secified as the second argument.
  # bash ecr.sh push foo 1.1
  ecr_push $2 $3
else
  # Push the repo secified as the second argument.
  # bash ecr.sh foo 1.1
  ecr_find_or_create $1
fi
