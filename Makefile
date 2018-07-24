BUILD_DIR = bin
BUILD_DOCKER_IMAGE = boruta-build-img
BUILD_DOCKER_CONTAINER = boruta-build

.PHONY: all
all: docker-build

.PHONY: clean
clean: clean-docker-build

.PHONY: docker-build
docker-build: docker-build-img
	mkdir -p "${BUILD_DIR}"
	docker create --name "${BUILD_DOCKER_CONTAINER}" "${BUILD_DOCKER_IMAGE}"
	docker cp "${BUILD_DOCKER_CONTAINER}:/boruta" "${BUILD_DIR}/boruta"
	docker cp "${BUILD_DOCKER_CONTAINER}:/dryad_armv7" "${BUILD_DIR}/dryad_armv7"
	docker cp "${BUILD_DOCKER_CONTAINER}:/dryad_amd64" "${BUILD_DIR}/dryad_amd64"
	docker rm "${BUILD_DOCKER_CONTAINER}"

.PHONY: docker-build-img
docker-build-img:
	docker build --tag "${BUILD_DOCKER_IMAGE}" .

.PHONY: clean-docker-build
clean-docker-build:
	-docker rm "${BUILD_DOCKER_CONTAINER}"
	-docker rmi "${BUILD_DOCKER_IMAGE}"
