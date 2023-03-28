//go:build gcs_migrator
// +build gcs_migrator

// For testing gcs-migrator locally
// Run below commands from gcs-migrator directory:
//
//		# requires gsutil, python and pip installed, set GOOGLE_APPLICATION_CREDENTIALS env or run
//		gcloud auth application-default login
//		export GOOGLE_CLOUD_PROJECT=dev-common-rsc-68w56 # or any other project-id
//	 # run test, additional flags can be passed to test
//		go test -v gcs-migrator_test.go --failures
//
// Package gcs_migrator
package gcs_migrator

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/estafette/estafette-ci-api/pkg/migrationpb"
)

const objectsInBucket = "objects.txt"

var (
	serverAddr        = flag.String("addr", "localhost:50051", "The server address in the format of host:port")
	migrationCount    = flag.Int("migration-count", 10, "Number of objects to migrate in test")
	forceFetchObjects = flag.Bool("force-fetch-objects", false, "Force fetching objects list from source bucket")
	srcPath           = flag.String("src-path", "logs/bitbucket.org/xivart/origami/builds", "Source path to migrate objects from")
	srcBucket         = flag.String("src-bucket", "estafette-ci-dev-api-dev-common-rsc", "Source bucket to migrate objects from")
	destBucket        = flag.String("dest-bucket", "gcs-test-migration", "Destination bucket to migrate objects in")
	destPath          = flag.String("dest-path", "logs/github.com/xivart/origami/builds", "Destination path to migrate objects in")
	failures          = flag.Bool("failures", false, "Fail 10% objects in migration by providing invalid source path")
)

func startServer(t *testing.T) *exec.Cmd {
	t.Log("Installing requirements for gcs-migrator")
	cmd := exec.Command("pip", "install", "-r", "requirements.txt")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to install gcs-migrator requirements: %v", err)
	}
	t.Logf("Getting objects list from gs://%s/%s", *srcBucket, *srcPath)
	if *forceFetchObjects {
		_ = os.Remove(objectsInBucket)
	}
	if _, err := os.Stat(objectsInBucket); errors.Is(err, os.ErrNotExist) {
		t.Logf("fetching objects list from gs://%s/%s", *srcBucket, *srcPath)
		cmd = exec.Command("bash", "-c", fmt.Sprintf(`gsutil ls -r gs://%s/%s | grep -Ei '^.*\.log$' | rev | cut -d '/' -f 1 | rev > objects.txt `, *srcBucket, *srcPath))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			t.Fatalf("Failed to get objects list from gs://%s/%s: %v", *srcBucket, *srcPath, err)
		}
	} else {
		t.Logf("objects.txt exists, skipped fetching objects list from gs://%s/%s", *srcBucket, *srcPath)
	}
	t.Log("Starting gcs-migrator server")
	cmd = exec.Command("python", "server.py")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start gcs-migrator server: %v", err)
	}
	time.Sleep(5 * time.Second)
	return cmd
}

func newRequest(from, to string) *migrationpb.Request {
	return &migrationpb.Request{
		FromLogObject: &migrationpb.LogObject{
			Bucket: *srcBucket,
			Path:   *srcPath + "/" + from,
		},
		ToLogObject: &migrationpb.LogObject{
			Bucket: *destBucket,
			Path:   *destPath + "/" + to,
		},
	}
}

func TestGcsMigrator(t *testing.T) {
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
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"messageuniqueid": "gcs-migrator-test"}))
	var objects *os.File
	if objects, err = os.Open(objectsInBucket); err != nil {
		t.Errorf("Failed to open objects.txt")
	}
	defer func(file *os.File) {
		if err = file.Close(); err != nil {
			t.Errorf("Failed to close objects.txt: %v", err)
		}
	}(objects)
	scanner := bufio.NewScanner(objects)
	count := 1
	expectedFailures := 0
	client := migrationpb.NewServiceClient(conn)
	var stream migrationpb.Service_MigrateClient
	if stream, err = client.Migrate(ctx, grpc.MaxCallRecvMsgSize(100*1024*1024)); err != nil {
		t.Fatalf("client.Migrate creation failed: %v", err)
	}
	for scanner.Scan() {
		if count > *migrationCount {
			break
		}
		req := newRequest(scanner.Text(), fmt.Sprintf("%d.log", count))
		if *failures && count%10 == 0 {
			expectedFailures++
			req.FromLogObject.Path = "path-should-not-exist-in-this-bucket"
		}
		if err = stream.Send(req); err != nil {
			t.Fatalf("client.Migrate: stream.Send(%v) failed: %v", req, err)
		}
		count++
	}
	var response *migrationpb.Response
	if response, err = stream.CloseAndRecv(); err != nil {
		t.Fatalf("client.Migrate: stream.CloseAndRecv() failed: %v", err)
	}
	actualFailures := 0
	for _, migration := range response.Migrations {
		if migration.Error != nil {
			actualFailures++
		}
	}
	t.Logf("completed in %v elapsed", time.Since(start))
	if expectedFailures != actualFailures {
		t.Errorf("Expected %d failures, got %d", expectedFailures, actualFailures)
	}
	_ = server.Process.Kill()
	time.Sleep(1 * time.Second)
}
