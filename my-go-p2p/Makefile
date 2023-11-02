build:
	go build -o bin/app

run: build
	./bin/app


run_n1: build
	./bin/app --name node_n1 --port 8287

run_n2: build
	./bin/app --name node_n2 --port 8288

run_n3: build
	./bin/app --name node_n3 --port 8289

run_n4: build
	./bin/app --name node_n4 --port 8290


test:
	go test -v ./...
