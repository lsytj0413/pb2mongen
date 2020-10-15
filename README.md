# pb2mongen

# How to use

1. git clone git@github.com:lsytj0413/pb2mongen.git
2. cd pb2mongen
3. go build -o protoc-gen-go ./protoc-gen-mgo/main.go
4. protoc --go_out=./example --plugin=protoc-gen-go={PATH}/pb2mongen/protoc-gen-go  ./example/*.proto