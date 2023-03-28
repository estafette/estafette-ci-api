package gcs_migrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/estafette/estafette-ci-api/pkg/migrationpb"
)

func TestHealthCheckResponse(t *testing.T) {
	serverAddr := "localhost:50051"
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		t.Errorf("failed to dial grpc connection with gcs-migrator: %v", err)
	}
	client := migrationpb.NewHealthClient(conn)
	_, err = client.Check(context.Background(), &migrationpb.HealthCheckRequest{})
	if !assert.ErrorContains(t, err, "50051: connect: connection refused") {
		t.Fatalf("grpc returned invalid error for health unavailable: %v", err)
	}
}
