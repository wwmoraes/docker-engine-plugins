ARCH ?= $(shell go env GOARCH)
TAG ?= latest
GO_VERSION ?= 1.18
ALPINE_VERSION ?= 3.15

REPOSITORY = wwmoraes

ARCHITECTURES = armv7l aarch64 x86_64
PLATFORM_armv7l := linux/arm/v7
PLATFORM_aarch64 := linux/arm64
PLATFORM_x86_64 := linux/amd64
# Apple M1
PLATFORM_arm64 := linux/arm64

PLUGINS := $(patsubst %/,%,$(dir $(wildcard */Dockerfile)))
PLATFORM = ${PLATFORM_${ARCH}}

all: ${PLUGINS}

.PHONY: clean
clean: $(addprefix clean-,${PLUGINS})

.PHONY: rmp
rmp: $(addprefix rmp-,${PLUGINS})

.PHONY: rmi
rmi: $(addprefix rmi-,${PLUGINS})

.PHONY: clean-%
clean-%: %/
	@${RM} -rf $</build*
	@${RM} -rf $</build*

.PHONY: rmp-%
rmp-%: %/
	@docker plugin rm -f ${REPOSITORY}/$<:${ARCH}-${TAG} || true

.PHONY: rmi-%
rmi-%: %/
	@docker rmi -f $<:${DOCKER_TAG} || true

.PHONY: ${PLUGINS}
${PLUGINS}: %: %/Dockerfile %/config.json $(wildcard %/*.go)
${PLUGINS}: DOCKER_TAG=${ARCH}-${TAG}
${PLUGINS}:
ifeq (${PLATFORM},)
	$(error Architecture ${ARCH} not supported. Set PLATFORM_${ARCH} and try again.)
endif
	$(info building $@ for ${ARCH} (${PLATFORM})...)
	@docker plugin rm -f ${REPOSITORY}/$@:${ARCH}-${TAG} 2>/dev/null || true
	@docker rmi -f $@:${DOCKER_TAG} 2>/dev/null || true
	@docker buildx build --load --platform ${PLATFORM} \
		--build-arg ALPINE_VERSION=${ALPINE_VERSION} \
		--build-arg GO_VERSION=${GO_VERSION} \
		-t $@:${DOCKER_TAG} -f $@/Dockerfile .
	@docker create --name ${DOCKER_TAG} --platform ${PLATFORM} $@:${DOCKER_TAG}
	@rm -rf $@/build-${ARCH}/rootfs
	@mkdir -p $@/build-${ARCH}/rootfs
	@docker export ${DOCKER_TAG} | tar -x -C $@/build-${ARCH}/rootfs
	@docker rm -vf ${DOCKER_TAG}
	@cp $@/config.json $@/build-${ARCH}
	@docker plugin create ${REPOSITORY}/$@:${ARCH}-${TAG} $@/build-${ARCH}

run-%: %
	$(info running $*-dev)
	@docker run -it -d \
		--name $*-dev \
		--env-file $*/.env.local \
		-v "${PWD}/$*/token:/run/secrets/op/token:ro" \
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

enable-op-secret-plugin: op-secret-plugin/.env.local
	@source $< && docker plugin set ${REPOSITORY}/op-secret-plugin:${ARCH}-${TAG} \
		OP_CONNECT_HOST=$${OP_CONNECT_HOST} \
		OP_CONNECT_TOKEN="$${OP_CONNECT_TOKEN}"
	@docker plugin enable ${REPOSITORY}/op-secret-plugin:${ARCH}-${TAG}

build-%:
	@$(foreach ARCH,${ARCHITECTURES},${MAKE} $* ARCH=${ARCH};)

release-%:
	@$(foreach ARCH,${ARCHITECTURES},docker plugin push ${REPOSITORY}/$*:${ARCH}-${TAG};)
