/*
 * Copyright (c) 2019. Rendall messenger.
 */

//go:generate protoc -I=. --proto_path=. --gofast_out=. api.proto
//go:generate rendall-rpc-code-gen -proto=api.proto
package sessionInternalAPI
