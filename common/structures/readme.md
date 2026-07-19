to recompile the protos use this

$ export PATH=$PATH:/home/opencnc/go/bin

$ protoc -I=.   --go_out=. --go_opt=paths=source_relative   ./common/structures/topology_config/topology_config.proto


for the grpc part
protoc -I=.   --go-grpc_out=. --go-grpc_opt=paths=source_relative   ./common/structures/service/service.proto