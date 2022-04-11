all:
	for cmd in $(patsubst cmd/%,%,$(wildcard cmd/*)); do \
		${GO_OPTS} go build -o bin/$$cmd cmd/$$cmd/main.go; \
	done

.PHONY: clean tools
clean:
	@rm -rf bin

.DEFAULT_GOAL := all
