package gnmi

// Tests SHOW bgp running-config

import (
	"crypto/tls"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

func TestGetBGPRunningConfig(t *testing.T) {
	s := createServer(t, ServerPort)
	go runServer(t, s)
	defer s.ForceStop()

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	opts := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))}

	conn, err := grpc.Dial(TargetAddr, opts...)
	if err != nil {
		t.Fatalf("Dialing to %q failed: %v", TargetAddr, err)
	}
	defer conn.Close()

	gClient := pb.NewGNMIClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout*time.Second)
	defer cancel()

	mockOutputFile := "../testdata/VTYSH_SHOW_BGP_RUNNING_CONFIG.txt"
	mockOutput, err := os.ReadFile(mockOutputFile)
	if err != nil {
		t.Fatalf("Reading %q failed: %v", mockOutputFile, err)
	}
	wantRunningConfig, err := json.Marshal(string(mockOutput))
	if err != nil {
		t.Fatalf("Marshaling mock output failed: %v", err)
	}

	tests := []struct {
		desc           string
		wantRetCode    codes.Code
		wantRespVal    interface{}
		valTest        bool
		mockOutputFile string
	}{
		{
			desc:        "query SHOW bgp running-config read error",
			wantRetCode: codes.NotFound,
		},
		{
			desc:           "query SHOW bgp running-config",
			wantRetCode:    codes.OK,
			wantRespVal:    wantRunningConfig,
			valTest:        true,
			mockOutputFile: mockOutputFile,
		},
	}

	textPbPath := `
		elem: <name: "bgp" >
		elem: <name: "running-config" >
	`
	for _, test := range tests {
		var patches *gomonkey.Patches
		if test.mockOutputFile != "" {
			patches = MockNSEnterCommand(t, test.mockOutputFile)
		}

		t.Run(test.desc, func(t *testing.T) {
			runTestGet(t, ctx, gClient, "SHOW", textPbPath, test.wantRetCode, test.wantRespVal, test.valTest)
		})
		if patches != nil {
			patches.Reset()
		}
	}
}
