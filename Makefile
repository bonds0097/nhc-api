SHELL=/bin/bash

local: docker_compose

docker_compose:
	docker-compose build
	docker-compose run --service-ports nhc-api