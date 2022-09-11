-include .env.local
export

GO = grc go

include .make/*.mk

coverage: golang-coverage
test: golang-test

ARCH ?= $(shell go env GOARCH)

ifeq (${ARCH},aarch64)
ARCH = arm64
endif

REPOSITORY = wwmoraes

ARCHITECTURES = armv7l arm64 x86_64
PLATFORM_armv7l := linux/arm/v7
PLATFORM_arm64 := linux/arm64
PLATFORM_x86_64 := linux/amd64
# Apple M1
PLATFORM_arm64 := linux/arm64

PLUGINS := $(patsubst cmd/%/Dockerfile,%,$(wildcard cmd/*/Dockerfile))
PLATFORM = ${PLATFORM_${ARCH}}

all: ${PLUGINS}

.PHONY: clean
clean: $(addprefix clean-,${PLUGINS})
	${RM} -r bin
	${RM} -r coverage

.PHONY: rmp
rmp: $(addprefix rmp-,${PLUGINS})

.PHONY: rmi
rmi: $(addprefix rmi-,${PLUGINS})

.PHONY: clean-%
clean-%:
	${RM} -r $*/build*

.PHONY: rmp-%
rmp-%: %/
	@docker plugin rm -f ${REPOSITORY}/$<:${ARCH}-${TAG} || true

.PHONY: rmi-%
rmi-%: cmd/%
	@docker rmi -f $*:${DOCKER_TAG} || true

# TODO scoped sources
.PHONY: ${PLUGINS}
${PLUGINS}: DOCKER_TAG=${ARCH}-${TAG}
${PLUGINS}: %: cmd/%/Dockerfile cmd/%/config.json ${GOLANG_SOURCE_FILES}
ifeq (${PLATFORM},)
	$(error Architecture ${ARCH} not supported. Set PLATFORM_${ARCH} and try again.)
endif
	$(info building $* for ${ARCH} (${PLATFORM})...)
	@docker plugin rm -f ${REPOSITORY}/$@:${ARCH}-${TAG} 2>/dev/null || true
	@docker rmi -f $@:${DOCKER_TAG} 2>/dev/null || true
	@docker buildx build --load --platform ${PLATFORM} \
		--build-arg ALPINE_VERSION=${ALPINE_VERSION} \
		--build-arg GO_VERSION=${GO_VERSION} \
		-t $@:${DOCKER_TAG} -f cmd/$@/Dockerfile .
	@docker create --name ${DOCKER_TAG} --platform ${PLATFORM} $@:${DOCKER_TAG}
	@rm -rf cmd/$@/build-${ARCH}/rootfs
	@mkdir -p cmd/$@/build-${ARCH}/rootfs
	@docker export ${DOCKER_TAG} | tar -x -C cmd/$@/build-${ARCH}/rootfs
	@docker rm -vf ${DOCKER_TAG}
	@cp cmd/$@/config.json cmd/$@/build-${ARCH}
	@docker plugin create ${REPOSITORY}/$@:${ARCH}-${TAG} cmd/$@/build-${ARCH}

run-%: %
	$(info running $*-dev)
	@docker run -it -d \
		--name $*-dev \
		--env-file cmd/$*/.env.local \
		-v "${PWD}/cmd/$*/token:/run/secrets/op/token:ro" \
		$*:${ARCH}-${TAG}
	@docker exec -it $*-dev apk update
	@docker exec -it $*-dev apk add curl
	-@docker attach $*-dev
	@${MAKE} clean-run-$*

clean-run-%:
	@docker container rm $*-dev

test-op-secret-plugin:
	@docker exec -it op-secret-plugin-dev curl -X POST \
		--unix-socket /run/docker/plugins/op.sock \
		-d '{"SecretName":"grafana","SecretLabels":{"connect.1password.io/vault":"Lab","connect.1password.io/item":"Grafana","connect.1password.io/field":"n"}}' \
		http://localhost/SecretProvider.GetSecret | jq .

enable-op-secret-plugin: cmd/op-secret-plugin/.env.local
	@source $< && docker plugin set ${REPOSITORY}/op-secret-plugin:${ARCH}-${TAG} \
		OP_CONNECT_HOST=$${OP_CONNECT_HOST} \
		OP_CONNECT_TOKEN="$${OP_CONNECT_TOKEN}"
	@docker plugin enable ${REPOSITORY}/op-secret-plugin:${ARCH}-${TAG}

disable-%:
	@docker plugin enable ${REPOSITORY}/$@:${ARCH}-${TAG}

build-%:
	@$(foreach ARCH,${ARCHITECTURES},${MAKE} $* ARCH=${ARCH};)

rm-%:
	-@$(foreach ARCH,${ARCHITECTURES},docker plugin disable ${REPOSITORY}/$*:${ARCH}-${TAG};)
	-@$(foreach ARCH,${ARCHITECTURES},docker plugin rm ${REPOSITORY}/$*:${ARCH}-${TAG};)

release-%:
	@$(foreach ARCH,${ARCHITECTURES},docker plugin push ${REPOSITORY}/$*:${ARCH}-${TAG};)

diff-vendor: GOPATH=$(shell go env GOPATH)
diff-vendor:
	@diff --unidirectional-new-file -r -u \
		${GOPATH}/pkg/mod/github.com/cch123/supermonkey@v1.0.1 \
		vendor/github.com/cch123/supermonkey \
		| grep -vE "^Only in" | sed -E "s|^(diff ?.*) ${GOPATH}|\1 \$$GOPATH|g" | sed -E "s|^--- ${GOPATH}|--- \$$GOPATH|g" > vendor.patch

golang-test: GOARCH=amd64
golang-test: GOLANG_FLAGS=-race -ldflags="-s=false" -gcflags=all=-l

golang-coverage: GOARCH=amd64
golang-coverage: GOLANG_FLAGS=-race -ldflags="-s=false" -gcflags=all=-l

test-secret:
	docker secret create -d ${REPOSITORY}/op-secret-plugin:${ARCH}-${TAG} \
		-l connect.1password.io/vault=Lab \
		-l connect.1password.io/item=Grafana \
		-l connect.1password.io/field=k \
	test
