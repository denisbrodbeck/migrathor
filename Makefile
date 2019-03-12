.PHONY: default db-start db-init db-reset db-clean

ENGINE := docker
NAME := migrathor

# use docker and fallback to podman
ifeq (, $(shell which docker 2> /dev/null))
	ENGINE = podman
endif

default: db-start db-init

db-start:
	@echo starting $(NAME) container
	sudo $(ENGINE) run -d -p 5432:5432 --name $(NAME) postgres:11-alpine

db-init:
	@echo init schema
	@sleep 5 # give pg container time to boot
	sudo $(ENGINE) exec -it $(NAME) psql -U postgres -c "REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC; DROP SCHEMA public CASCADE; CREATE SCHEMA public;" postgres
#sudo $(ENGINE) exec -it $(NAME) psql -U postgres -c "REVOKE ALL PRIVILEGES ON SCHEMA public FROM PUBLIC; DROP SCHEMA public CASCADE; CREATE ROLE $(NAME) WITH LOGIN PASSWORD 'secret'; CREATE SCHEMA AUTHORIZATION $(NAME);" postgres

db-clean:
	@echo removing $(NAME) container
	@sudo $(ENGINE) rm $(NAME) --force

db-reset: db-clean db-start db-init
