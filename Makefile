.PHONY: build clean image

build:
	@ci/scripts/build-code.sh

clean: clean-images

clean-images:
	@docker images \
		| grep -iF 'rattab/gups' \
		| awk '{ print $$1":"$$2 }' \
		| xargs -I {} docker rmi {}

image:
	@ci/scripts/build-image.sh
