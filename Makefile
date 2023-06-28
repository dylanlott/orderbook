test:	
	go test -race -count 1 -v ./pkg/... 

test-benchmark:
	go test -benchtime=60s ./pkg/orderbook/...

test-profile: 
	go test -cpuprofile=prof.out ./pkg/orderbook/...

build-docker:
	docker build -t golem .

run-docker: build-docker
	docker run -it -p 1323:1323 golem

deploy: test
	./deploy.sh