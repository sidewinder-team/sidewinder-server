
# make sure build tools are installed
EXECUTABLES = docker docker-compose
OK := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),,$(error "No '$(exec)' in PATH)))

TEST_COMPOSE = docker-compose -f docker-compose-tdd.yml

# ----------------------------------------------------------------------

tdd: docker-connected test-clean
	$(TEST_COMPOSE) up

docker-connected:
	docker ps &>/dev/null

test-clean:
	$(TEST_COMPOSE) kill
	$(TEST_COMPOSE) rm --force
