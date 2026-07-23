package api

//go:generate oapi-codegen -generate types -package api -o types.gen.go ../../../docs/api.yaml
//go:generate oapi-codegen -generate chi-server -package api -o server.gen.go ../../../docs/api.yaml
//go:generate oapi-codegen -generate types,client -package client -o ../../../client/client.gen.go ../../../docs/api.yaml
