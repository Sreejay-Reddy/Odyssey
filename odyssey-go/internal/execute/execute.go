package execute 

import (
	"context"
)

type execute struct {
	key string
	target string
	ttlMS string
}

func (e *execute) execute(ctx context.Context){}
