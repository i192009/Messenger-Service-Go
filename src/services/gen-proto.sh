#!/bin/bash

pwd
files="MessengerService.proto UserService.proto ZixelBus.proto thirdAdapter.proto"
    
for file in $files; do
    echo $file
    protoc --proto_path=../grpc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --experimental_allow_proto3_optional ../grpc/$file
done
