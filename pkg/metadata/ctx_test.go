package metadata

import (
	"context"
	"fmt"
	"testing"
)

func TestCtx(t *testing.T) {
	ctx := context.Background()
	//a.Set("aaa", "vwvwev")
	ctx1 := Set(ctx, "aaa", "zzz")
	ctx2 := Set(ctx1, "bbb", "ddd")
	ctx3 := Set(ctx2, "bbb", "333")

	fmt.Println("ctx1 aaa")
	fmt.Println(Get(ctx1, "aaa"))
	fmt.Println("ctx2 aaa")
	fmt.Println(Get(ctx2, "aaa"))
	fmt.Println("ctx1 bbb")
	fmt.Println(Get(ctx1, "bbb"))
	fmt.Println("ctx2 bbb")
	fmt.Println(Get(ctx2, "bbb"))
	fmt.Println("ctx3 bbb")
	fmt.Println(Get(ctx3, "bbb"))
}
