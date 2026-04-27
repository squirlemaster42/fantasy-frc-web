package types

import "context"

type FaroData struct {
	Token string
	Nonce string
}

type faroDataKey struct{}

var FaroContextKey = faroDataKey{}

func GetFaroData(ctx context.Context) FaroData {
	if fd, ok := ctx.Value(FaroContextKey).(FaroData); ok {
		return fd
	}
	return FaroData{}
}
