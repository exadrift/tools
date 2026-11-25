.PHONY: run
run: build-one
	if [ "$$ARGS" = "" ]; then \
		bin/$(NAME)-$(OS)-$(ARCH); \
	else \
		bin/$(NAME)-$(OS)-$(ARCH) $$ARGS; \
	fi