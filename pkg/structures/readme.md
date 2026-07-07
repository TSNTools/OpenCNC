to recompile the protos use this

$ export PATH=$PATH:/home/opencnc/go/bin

$ protoc -I=.   --go_out=. --go_opt=paths=source_relative   ./pkg/structures/topology_config/topology_config.proto