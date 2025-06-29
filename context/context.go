package context

import "context"

type (
	TagKey struct{}
)

var keyTagName = TagKey{}

func GetTagName(ctx context.Context) string {
	if v := ctx.Value(keyTagName); v != nil {
		return v.(string)
	}
	return ""
}

func SetTagName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, keyTagName, name)
}
