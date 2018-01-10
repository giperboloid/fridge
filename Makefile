build:
	protoc  -I api/pb/ api/pb/api.proto \
		-I/usr/local/include \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:api/pb \
	
	GOOS=linux GOARCH=amd64 go build
	docker build -t fridgems .
	
run: 
	docker run -p 50051:50051 fridgems \
	-e CENTER_TCP_ADDR=127.0.0.1 \
	-e CENTER_DATA_TCP_PORT=3126 \
	-e CENTER_CONFIG_TCP_PORT=3092 \
	
