package middlewares

import (
	"context"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryLogger(log zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Info().
			Str("method", info.FullMethod).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("grpc unary request")
		return resp, err
	}
}

func StreamLogger(log zerolog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, stream)
		log.Info().
			Str("method", info.FullMethod).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("grpc stream request")
		return err
	}
}

func UnaryRateLimit(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

func StreamRateLimit(limiter *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !limiter.Allow() {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(srv, stream)
	}
}

func UnaryAuth(validator TokenValidator, allowMethods map[string]struct{}) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, ok := allowMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}
		token := extractGRPCToken(ctx)
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "authorization required")
		}
		userID, err := validator.ValidateSession(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		ctx = ContextWithUserID(ctx, userID)
		return handler(ctx, req)
	}
}

func StreamAuth(validator TokenValidator, allowMethods map[string]struct{}) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if _, ok := allowMethods[info.FullMethod]; ok {
			return handler(srv, stream)
		}
		token := extractGRPCToken(stream.Context())
		if token == "" {
			return status.Error(codes.Unauthenticated, "authorization required")
		}
		userID, err := validator.ValidateSession(token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}
		ctx := ContextWithUserID(stream.Context(), userID)
		wrapped := &wrappedStream{ServerStream: stream, ctx: ctx}
		return handler(srv, wrapped)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func RequireTLS() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !hasTLS(ctx) {
			return nil, status.Error(codes.PermissionDenied, "TLS required")
		}
		return handler(ctx, req)
	}
}

func RequireTLSStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !hasTLS(stream.Context()) {
			return status.Error(codes.PermissionDenied, "TLS required")
		}
		return handler(srv, stream)
	}
}

func extractGRPCToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	authHeader := values[0]
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
}

func hasTLS(ctx context.Context) bool {
	p, ok := peer.FromContext(ctx)
	if !ok || p.AuthInfo == nil {
		return false
	}
	_, ok = p.AuthInfo.(credentials.TLSInfo)
	return ok
}
