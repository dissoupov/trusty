package client_test

import (
	"context"
	"testing"

	v1 "github.com/go-phorce/trusty/api/v1"
	pb "github.com/go-phorce/trusty/api/v1/trustypb"
	"github.com/go-phorce/trusty/client"
	"github.com/go-phorce/trusty/client/proxy"
	"github.com/go-phorce/trusty/tests/mockpb"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusWithNewCtxClient(t *testing.T) {
	ctx := context.Background()
	srv := &mockpb.MockStatusServer{}

	cli := client.NewCtxClient(ctx)
	cli.Status = client.NewStatusFromProxy(proxy.StatusServerToClient(srv))
	defer cli.Close()

	vexp := &pb.ServerVersion{Build: "1234", Runtime: "go1.15"}
	srv.Resps = []proto.Message{vexp}
	vres, err := cli.Version(ctx)
	require.NoError(t, err)
	assert.Equal(t, *vexp, *vres)

	sexp := &pb.ServerStatusResponse{
		Status: &pb.ServerStatus{
			Name: "test",
		},
		Version: vexp,
	}
	srv.Resps = []proto.Message{sexp}
	sres, err := cli.Server(ctx)
	require.NoError(t, err)
	assert.Equal(t, *sexp, *sres)

	cexp := &pb.CallerStatusResponse{
		Id:   "1234",
		Name: "denis",
		Role: "admin",
	}
	srv.Resps = []proto.Message{cexp}
	cres, err := cli.Caller(ctx)
	require.NoError(t, err)
	assert.Equal(t, *cexp, *cres)
}

func TestStatusWithNewClientMock(t *testing.T) {
	ctx := context.Background()
	srv := &mockpb.MockStatusServer{}

	cli, grpc := setupStatusMockGRPC(t, srv)
	defer grpc.Stop()
	defer cli.Close()

	assert.NotNil(t, cli.ActiveConnection())

	expErr := v1.ErrGRPCPermissionDenied

	t.Run("Version", func(t *testing.T) {
		vexp := &pb.ServerVersion{Build: "1234", Runtime: "go1.15"}
		srv.SetResponse(vexp)
		vres, err := cli.Version(ctx)
		require.NoError(t, err)
		assert.Equal(t, *vexp, *vres)

		srv.Err = expErr
		_, err = cli.Version(ctx)
		require.Error(t, err)
		assert.Equal(t, expErr.Error(), err.Error())
	})

	t.Run("ServerStatus", func(t *testing.T) {
		sexp := &pb.ServerStatusResponse{
			Status: &pb.ServerStatus{
				Name: "test",
			},
		}
		srv.SetResponse(sexp)
		sres, err := cli.Server(ctx)
		require.NoError(t, err)
		assert.Equal(t, *sexp, *sres)

		srv.Err = expErr
		_, err = cli.Server(ctx)
		require.Error(t, err)
		assert.Equal(t, expErr.Error(), err.Error())
	})

	t.Run("CallerStatus", func(t *testing.T) {
		cexp := &pb.CallerStatusResponse{
			Role: "admin",
		}
		srv.SetResponse(cexp)
		cres, err := cli.Caller(ctx)
		require.NoError(t, err)
		assert.Equal(t, *cexp, *cres)

		srv.Err = expErr
		_, err = cli.Caller(ctx)
		require.Error(t, err)
		assert.Equal(t, expErr.Error(), err.Error())
	})
}
