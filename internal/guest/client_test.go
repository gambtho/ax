// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package guest

import (
	"context"
	"net"
	"testing"
	"time"

	ateenvv1alpha "github.com/agent-substrate/env/proto/ateenv/v1alpha"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type authorityProcessServer struct {
	ateenvv1alpha.UnimplementedProcessServiceServer
	authority chan string
}

func (s *authorityProcessServer) StartProcess(ctx context.Context, _ *ateenvv1alpha.StartProcessRequest) (*ateenvv1alpha.Process, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(":authority")
	if len(values) > 0 {
		s.authority <- values[0]
	}
	return &ateenvv1alpha.Process{ProcessId: "test"}, nil
}

func TestDialTargetUsesActorDNSAuthority(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	srv := grpc.NewServer()
	process := &authorityProcessServer{authority: make(chan string, 1)}
	ateenvv1alpha.RegisterProcessServiceServer(srv, process)
	go srv.Serve(lis)
	defer srv.Stop()

	client, err := DialTarget(lis.Addr().String(), "hello/hello-fixed-greeting")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := client.process.StartProcess(ctx, &ateenvv1alpha.StartProcessRequest{}); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-process.authority:
		if want := "hello-fixed-greeting.hello.actors.resources.substrate.ate.dev"; got != want {
			t.Fatalf("authority = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive gRPC authority")
	}
}
