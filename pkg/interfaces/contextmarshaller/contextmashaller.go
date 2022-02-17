package contextmarshaller

import "context"

type ContextMarshaller interface {
	Marshal(ctx context.Context) ([]byte, error)
	Unmarshal([]byte) (context.Context, context.CancelFunc, error)
}
