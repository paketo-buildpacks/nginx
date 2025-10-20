.PHONY: test retrieve

id:
	@echo nginx

test:
	@cd test; \
	./run-test --version $(version) --tarballPath $(tarballPath) --os $(os) --arch $(arch)

retrieve:
	@cd retrieval; \
	go run . \
		--buildpack_toml_path=$(buildpackTomlPath) \
		--output=$(output)
