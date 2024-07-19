// Copyright 2022 Lekko Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"sync"
	"time"

	"github.com/lekkodev/go-sdk/pkg/debug"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func makeProvider(
	rp *retryProvider,
	delay time.Duration,
	f func(
		context.Context,
		*RepositoryKey,
		...ProviderOption,
	) (Provider, error),
	ctx context.Context,
	rk *RepositoryKey,
	opts ...ProviderOption,
) {
	p, err := f(ctx, rk, opts...)
	if err != nil {
		if delay <= 60*time.Second {
			delay = time.Duration(float64(delay) * 1.5)
		}
		debug.LogInfo("Attempting to connect to Lekko", "delay", delay.Seconds())
		rp.lastError = err
		go makeProvider(rp, delay, f, ctx, rk, opts...)
		time.Sleep(delay)
	} else {
		rp.Lock()
		rp.inner = p
		rp.Unlock()
	}
}

func RetryProvider(f func(
	context.Context,
	*RepositoryKey,
	...ProviderOption,
) (Provider, error), ctx context.Context,
	rk *RepositoryKey,
	opts ...ProviderOption) (Provider, error) {
	rp := &retryProvider{}
	var p Provider
	p, err := f(ctx, rk, opts...)
	if err != nil {
		go makeProvider(rp, 5*time.Second, f, ctx, rk, opts...)
	}
	rp.inner = p
	rp.lastError = err
	return rp, err
}

type retryProvider struct {
	sync.RWMutex
	inner     Provider
	lastError error
}

func (p *retryProvider) Close(ctx context.Context) error {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.Close(ctx)
}

func (p *retryProvider) GetBool(ctx context.Context, key string, namespace string) (bool, error) {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return false, ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetBool(ctx, key, namespace)
}

func (p *retryProvider) GetFloat(ctx context.Context, key string, namespace string) (float64, error) {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return 0, ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetFloat(ctx, key, namespace)
}

func (p *retryProvider) GetInt(ctx context.Context, key string, namespace string) (int64, error) {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return 0, ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetInt(ctx, key, namespace)
}

func (p *retryProvider) GetJSON(ctx context.Context, key string, namespace string, result interface{}) error {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetJSON(ctx, key, namespace, result)
}

func (p *retryProvider) GetProto(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) error {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetProto(ctx, key, namespace, result)
}

func (p *retryProvider) GetString(ctx context.Context, key string, namespace string) (string, error) {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return "", ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetString(ctx, key, namespace)
}

func (p *retryProvider) GetAny(ctx context.Context, key string, namespace string) (protoreflect.ProtoMessage, error) {
	p.RLock()
	if p.inner == nil {
		p.RUnlock()
		return nil, ErrNoOpProvider
	}
	p.RUnlock()
	return p.inner.GetAny(ctx, key, namespace)
}
