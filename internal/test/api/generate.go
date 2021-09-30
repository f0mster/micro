//go:generate protoc --go_opt=paths=source_relative --go-vtproto_opt=features=marshal+unmarshal+size --go-vtproto_opt=paths=source_relative --go_out=. --go-vtproto_out=. ./*.proto
//go:generate micro-rpc-code-gen -proto=api.proto
package api
