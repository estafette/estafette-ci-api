//go:build gcs_migrator
// +build gcs_migrator

package gcs_migrator

import (
	"bufio"
	"context"
	"flag"
	"os"
	"os/exec"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/estafette/estafette-ci-api/pkg/migrationpb"
)

const objectsInBucket = "log_file_text.csv"

var (
	serverAddr = flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	srcBucket  = flag.String("src-bucket", "estafette-ci-dev-api-dev-common-rsc", "Source bucket to migrate objects from")
)

func startServer(t *testing.T) *exec.Cmd {
	t.Log("Starting gcs-migrator server")
	cmd := exec.Command("python", "server.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start gcs-migrator server: %v", err)
	}
	time.Sleep(5 * time.Second)
	return cmd
}

func newRequest(path string) *migrationpb.LogObject {
	return &migrationpb.LogObject{
		Bucket: *srcBucket,
		Path:   path,
	}
}

func TestGcsMigratorHealth(t *testing.T) {
	start := time.Now()
	t.Log("starting gcs-migrator test")
	flag.Parse()
	server := startServer(t)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		t.Fatalf("fail to dial connection with gcs-migrator: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		if err = conn.Close(); err != nil {
			t.Errorf("Failed to close connection with gcs-migrator: %v", err)
		}
	}(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var objects *os.File
	if objects, err = os.Open(objectsInBucket); err != nil {
		t.Errorf("Failed to open %s", objectsInBucket)
	}
	defer func(file *os.File) {
		if err = file.Close(); err != nil {
			t.Errorf("Failed to close objects.txt: %v", err)
		}
	}(objects)
	scanner := bufio.NewScanner(objects)
	count := 1
	client := migrationpb.NewLogCheckClient(conn)
	var stream migrationpb.LogCheck_CheckClient
	if stream, err = client.Check(ctx, grpc.MaxCallRecvMsgSize(100*1024*1024)); err != nil {
		t.Fatalf("client.Check creation failed: %v", err)
	}
	for scanner.Scan() {
		req := newRequest(scanner.Text())
		if err = stream.Send(req); err != nil {
			t.Fatalf("client.Check: stream.Send(%v) failed: %v", req, err)
		}
		count++
	}
	var response *migrationpb.LogCheckResponse
	if response, err = stream.CloseAndRecv(); err != nil {
		t.Fatalf("client.Migrate: stream.CloseAndRecv() failed: %v", err)
	}
	actualFailures := 0
	for _, logObject := range response.LogObjects {
		if !logObject.Exists {
			t.Errorf("%s - does not exist", logObject.LogObject.Path)
			actualFailures++
		}
	}
	t.Logf("completed in %v elapsed", time.Since(start))
	if actualFailures != 0 {
		t.Error("Some objects do not exist in the bucket")
	}
	_ = server.Process.Kill()
	time.Sleep(1 * time.Second)
}
