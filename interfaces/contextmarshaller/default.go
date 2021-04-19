package contextmarshaller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/f0mster/micro/pkg/metadata"
)

type DefaultCtxMarshaller struct {
}

type Data struct {
	Metadata metadata.Metadata
	Deadline *time.Time
}

func (d *DefaultCtxMarshaller) Marshal(ctx context.Context) ([]byte, error) {
	tmp := Data{}
	if ctx != nil {
		tmp.Metadata, _ = metadata.FromContext(ctx)
		dl, ok := ctx.Deadline()
		if ok {
			tmp.Deadline = &dl
		}
	}
	return json.Marshal(tmp)
}

func (d *DefaultCtxMarshaller) Unmarshal(data []byte) (context.Context, context.CancelFunc, error) {
	tmp := Data{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return nil, nil, err
	}

	if tmp.Deadline == nil || tmp.Deadline.IsZero() {
		ctx, cancel := context.WithCancel(context.Background())
		ctx = metadata.NewContext(ctx, tmp.Metadata)
		return ctx, cancel, nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), *tmp.Deadline)
	ctx = metadata.NewContext(ctx, tmp.Metadata)

	return ctx, cancel, nil
}
