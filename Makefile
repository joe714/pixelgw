BUILDROOT = $(PWD)
DOCKER_USERNAME ?= $(USER)
DOCKER_UGID = $(shell id $(USER) -u):$(shell id $(USER) -g)
DOCKER_USERFLAG = -u ${DOCKER_UGID}
DOCKER_NODE_IMAGE = node:22-alpine
APP_NAME = pixelgw
GIT_HASH ?= $(shell git log --format="%h" -n 1)

_BUILD_ARGS_TAG ?= ${GIT_HASH}
_BUILD_ARGS_RELEASE_TAG ?= latest

_STAGE1_IMAGE = ${DOCKER_USERNAME}/stage1:${GIT_HASH}
_DEPLOY_IMAGE = ${DOCKER_USERNAME}/${APP_NAME}:${_BUILD_ARGS_TAG}

_COMPOSE_DIR = deployments
_COMPOSE_PROJ ?= ${APP_NAME}
_COMPOSE_FILE ?= ${_COMPOSE_DIR}/compose.yaml
_COMPOSE_ENV_FILE ?= ${_COMPOSE_DIR}/env
_COMPOSE_TAG ?= ${GIT_HASH}

.PHONY: build generate deploy web


build: web
	docker build -f build/package/Dockerfile --tag ${DOCKER_USERNAME}/${APP_NAME}:${_BUILD_ARGS_TAG} .

stage1:
	docker build -f build/package/Dockerfile --target stage1 --tag ${_STAGE1_IMAGE} .

web:
	docker run --rm -it ${DOCKER_USERFLAG} -v .:/home/node -w /home/node/web ${DOCKER_NODE_IMAGE} \
	       	sh -c 'npm install && npm run build'

generate: stage1
	docker run --rm -it --user $$(id -u):$$(id -g) -v $$(pwd):/go/src ${_STAGE1_IMAGE} make -f build/Makefile generate

tag: build
	$(MAKE) _tag

deploy: deploy_test

deploy_%:
	$(MAKE) _deploy \
		-e _COMPOSE_PROJ=${APP_NAME}-$* \
		-e _COMPOSE_FILE=${_COMPOSE_DIR}/compose.$*.yaml \
		-e _COMPOSE_ENV_FILE=${_COMPOSE_DIR}/$*.env

all: build deploy

_tag:
	docker tag ${DOCKER_USERNAME}/${APP_NAME}:${_BUILD_ARGS_TAG} ${DOCKER_USERNAME}/${APP_NAME}:${_BUILD_ARGS_RELEASE_TAG}

_deploy:
	BUILDROOT=${BUILDROOT} \
	DOCKER_UGID=${DOCKER_UGID} \
	IMAGE=${DOCKER_USERNAME}/${APP_NAME}:${_COMPOSE_TAG} \
	docker compose \
		-p ${_COMPOSE_PROJ} \
		-f ${_COMPOSE_FILE} \
		--env-file ${_COMPOSE_ENV_FILE} \
		up --force-recreate --build -d
