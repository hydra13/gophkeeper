package middlewares

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/hydra13/gophkeeper/internal/models"
)

type testContextKey string

const wrappedStreamTestKey testContextKey = "test-key"

func TestUnaryAuth_PublicMethod(t *testing.T) {
	validator := &mockValidator{}
	allowMethods := map[string]struct{}{
		"/gophkeeper.v1.AuthService/Register": {},
	}

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.AuthService/Register"}
	_, err := UnaryAuth(validator, allowMethods)(context.Background(), nil, info, handler)
	require.NoError(t, err)
	require.True(t, called)
}

func TestUnaryAuth_NoToken(t *testing.T) {
	validator := &mockValidator{}
	allowMethods := map[string]struct{}{}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(context.Background(), nil, info, handler)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuth_ValidSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 42, nil
		},
	}
	allowMethods := map[string]struct{}{}

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		userID, ok := UserIDFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, int64(42), userID)
		return "ok", nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer valid-token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(ctx, nil, info, handler)
	require.NoError(t, err)
	require.True(t, called)
}

func TestUnaryAuth_RevokedSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionRevoked
		},
	}
	allowMethods := map[string]struct{}{}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer revoked-token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(ctx, nil, info, handler)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuth_ExpiredSession(t *testing.T) {
	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionExpired
		},
	}
	allowMethods := map[string]struct{}{}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer expired-token"))
	info := &grpc.UnaryServerInfo{FullMethod: "/gophkeeper.v1.DataService/GetRecord"}
	_, err := UnaryAuth(validator, allowMethods)(ctx, nil, info, handler)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ---------------------------------------------------------------------------
// UnaryLogger
// ---------------------------------------------------------------------------

func TestUnaryLogger_CallsHandler(t *testing.T) {
	t.Parallel()

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	resp, err := UnaryLogger(zerolog.Nop())(context.Background(), "req", info, handler)

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "response", resp)
}

func TestUnaryLogger_PropagatesError(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Error(codes.Internal, "boom")
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	_, err := UnaryLogger(zerolog.Nop())(context.Background(), nil, info, handler)

	require.Equal(t, codes.Internal, status.Code(err))
	require.Equal(t, "boom", status.Convert(err).Message())
}

// ---------------------------------------------------------------------------
// StreamLogger
// ---------------------------------------------------------------------------

func TestStreamLogger_CallsHandler(t *testing.T) {
	t.Parallel()

	var called bool
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test.StreamMethod"}
	stream := &mockServerStream{ctx: context.Background()}
	err := StreamLogger(zerolog.Nop())("srv", stream, info, handler)

	require.NoError(t, err)
	require.True(t, called)
}

func TestStreamLogger_PropagatesError(t *testing.T) {
	t.Parallel()

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return status.Error(codes.Unavailable, "stream broken")
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test.StreamMethod"}
	stream := &mockServerStream{ctx: context.Background()}
	err := StreamLogger(zerolog.Nop())("srv", stream, info, handler)

	require.Equal(t, codes.Unavailable, status.Code(err))
}

// ---------------------------------------------------------------------------
// UnaryRateLimit
// ---------------------------------------------------------------------------

func TestUnaryRateLimit_Allowed(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(5, time.Minute)

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	resp, err := UnaryRateLimit(limiter)(context.Background(), nil, info, handler)

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "ok", resp)
}

func TestUnaryRateLimit_Rejected(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(1, time.Minute)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	interceptor := UnaryRateLimit(limiter)

	// First request passes.
	_, err := interceptor(context.Background(), nil, info, handler)
	require.NoError(t, err)

	// Second request is rejected.
	_, err = interceptor(context.Background(), nil, info, handler)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "rate limit")
}

// ---------------------------------------------------------------------------
// StreamRateLimit
// ---------------------------------------------------------------------------

func TestStreamRateLimit_Allowed(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(5, time.Minute)

	var called bool
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test.StreamMethod"}
	stream := &mockServerStream{ctx: context.Background()}
	err := StreamRateLimit(limiter)("srv", stream, info, handler)

	require.NoError(t, err)
	require.True(t, called)
}

func TestStreamRateLimit_Rejected(t *testing.T) {
	t.Parallel()

	limiter := NewRateLimiter(1, time.Minute)

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test.StreamMethod"}
	stream := &mockServerStream{ctx: context.Background()}
	interceptor := StreamRateLimit(limiter)

	// First request passes.
	err := interceptor("srv", stream, info, handler)
	require.NoError(t, err)

	// Second request is rejected.
	err = interceptor("srv", stream, info, handler)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "rate limit")
}

// ---------------------------------------------------------------------------
// StreamAuth
// ---------------------------------------------------------------------------

func TestStreamAuth_PublicMethod(t *testing.T) {
	t.Parallel()

	validator := &mockValidator{}
	allowMethods := map[string]struct{}{
		"/gophkeeper.v1.AuthService/Register": {},
	}

	var called bool
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.AuthService/Register"}
	stream := &mockServerStream{ctx: context.Background()}
	err := StreamAuth(validator, allowMethods)("srv", stream, info, handler)

	require.NoError(t, err)
	require.True(t, called)
}

func TestStreamAuth_NoToken(t *testing.T) {
	t.Parallel()

	validator := &mockValidator{}
	allowMethods := map[string]struct{}{}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		t.Fatal("should not reach handler")
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.DataService/StreamRecords"}
	stream := &mockServerStream{ctx: context.Background()}
	err := StreamAuth(validator, allowMethods)("srv", stream, info, handler)

	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestStreamAuth_ValidSession(t *testing.T) {
	t.Parallel()

	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 99, nil
		},
	}
	allowMethods := map[string]struct{}{}

	var called bool
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		userID, ok := UserIDFromContext(stream.Context())
		require.True(t, ok)
		require.Equal(t, int64(99), userID)
		return nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer good-token"))
	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.DataService/StreamRecords"}
	stream := &mockServerStream{ctx: ctx}
	err := StreamAuth(validator, allowMethods)("srv", stream, info, handler)

	require.NoError(t, err)
	require.True(t, called)
}

func TestStreamAuth_InvalidToken(t *testing.T) {
	t.Parallel()

	validator := &mockValidator{
		validateSessionFn: func(token string) (int64, error) {
			return 0, models.ErrSessionRevoked
		},
	}
	allowMethods := map[string]struct{}{}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		t.Fatal("should not reach handler")
		return nil
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer bad-token"))
	info := &grpc.StreamServerInfo{FullMethod: "/gophkeeper.v1.DataService/StreamRecords"}
	stream := &mockServerStream{ctx: ctx}
	err := StreamAuth(validator, allowMethods)("srv", stream, info, handler)

	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// ---------------------------------------------------------------------------
// wrappedStream.Context
// ---------------------------------------------------------------------------

func TestWrappedStream_Context(t *testing.T) {
	t.Parallel()

	inner := &mockServerStream{ctx: context.Background()}
	customCtx := context.WithValue(context.Background(), wrappedStreamTestKey, "test-value")

	wrapped := &wrappedStream{ServerStream: inner, ctx: customCtx}

	require.Equal(t, customCtx, wrapped.Context())
	require.Equal(t, "test-value", wrapped.Context().Value(wrappedStreamTestKey))
}

// ---------------------------------------------------------------------------
// RequireTLS (unary)
// ---------------------------------------------------------------------------

func TestRequireTLS_WithTLS(t *testing.T) {
	t.Parallel()

	var called bool
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{},
	})
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	resp, err := RequireTLS()(ctx, nil, info, handler)

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, "ok", resp)
}

func TestRequireTLS_WithoutTLS(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	// No peer info in context → no TLS.
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	_, err := RequireTLS()(context.Background(), nil, info, handler)

	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "TLS required")
}

func TestRequireTLS_PeerWithoutAuthInfo(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Fatal("should not reach handler")
		return nil, nil
	}

	// Peer exists but has no AuthInfo.
	ctx := peer.NewContext(context.Background(), &peer.Peer{})
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Method"}
	_, err := RequireTLS()(ctx, nil, info, handler)

	require.Equal(t, codes.PermissionDenied, status.Code(err))
}

// ---------------------------------------------------------------------------
// RequireTLSStream
// ---------------------------------------------------------------------------

func TestRequireTLSStream_WithTLS(t *testing.T) {
	t.Parallel()

	var called bool
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		return nil
	}

	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{},
	})
	info := &grpc.StreamServerInfo{FullMethod: "/test.Stream"}
	stream := &mockServerStream{ctx: ctx}
	err := RequireTLSStream()("srv", stream, info, handler)

	require.NoError(t, err)
	require.True(t, called)
}

func TestRequireTLSStream_WithoutTLS(t *testing.T) {
	t.Parallel()

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		t.Fatal("should not reach handler")
		return nil
	}

	info := &grpc.StreamServerInfo{FullMethod: "/test.Stream"}
	stream := &mockServerStream{ctx: context.Background()}
	err := RequireTLSStream()("srv", stream, info, handler)

	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "TLS required")
}

// ---------------------------------------------------------------------------
// hasTLS
// ---------------------------------------------------------------------------

func TestHasTLS_NoPeer(t *testing.T) {
	t.Parallel()
	require.False(t, hasTLS(context.Background()))
}

func TestHasTLS_PeerNoAuthInfo(t *testing.T) {
	t.Parallel()
	ctx := peer.NewContext(context.Background(), &peer.Peer{})
	require.False(t, hasTLS(ctx))
}

func TestHasTLS_PeerWithTLSInfo(t *testing.T) {
	t.Parallel()
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{},
	})
	require.True(t, hasTLS(ctx))
}

func TestHasTLS_PeerWithNonTLSAuthInfo(t *testing.T) {
	t.Parallel()
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: fakeAuthInfo{},
	})
	require.False(t, hasTLS(ctx))
}

// ---------------------------------------------------------------------------
// extractGRPCToken (additional edge cases)
// ---------------------------------------------------------------------------

func TestExtractGRPCToken_NoMetadata(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", extractGRPCToken(context.Background()))
}

func TestExtractGRPCToken_EmptyAuthorization(t *testing.T) {
	t.Parallel()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	require.Equal(t, "", extractGRPCToken(ctx))
}

func TestExtractGRPCToken_NoBearerPrefix(t *testing.T) {
	t.Parallel()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Basic abc123"))
	require.Equal(t, "", extractGRPCToken(ctx))
}

func TestExtractGRPCToken_BearerWithSpaces(t *testing.T) {
	t.Parallel()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer  token-with-spaces "))
	require.Equal(t, "token-with-spaces", extractGRPCToken(ctx))
}

// ---------------------------------------------------------------------------
// mockServerStream — test helper for grpc.ServerStream
// ---------------------------------------------------------------------------

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

// fakeAuthInfo implements credentials.AuthInfo but is NOT TLSInfo.
type fakeAuthInfo struct{}

func (fakeAuthInfo) AuthType() string { return "fake" }
