V := @

.PHONY: vendor
vendor:
	$(V)go mod tidy
	$(V)go mod vendor
	$(V)git add vendor