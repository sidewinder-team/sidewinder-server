TEST_COMPOSE = docker-compose -f docker-compose-tdd.yml

# ----------------------------------------------------------------------

test: dependencies
	go test -v ./...

dependencies:
	go get -v -d -t

tdd: docker-connected test-clean
	$(TEST_COMPOSE) up

docker-connected:
	docker ps &>/dev/null

test-clean:
	$(TEST_COMPOSE) kill
	$(TEST_COMPOSE) rm --force
