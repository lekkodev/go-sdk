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
	"encoding/json"
	"fmt"

	"github.com/lekkodev/cli/pkg/encoding"
	"github.com/lekkodev/cli/pkg/feature"
	"github.com/lekkodev/cli/pkg/fs"
	"github.com/lekkodev/cli/pkg/metadata"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

func NewKubeProvider(kubeNamespace string) (Provider, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &kubeProvider{
		clientset.CoreV1().ConfigMaps(kubeNamespace),
	}, nil
}

type kubeProvider struct {
	kubeClient corev1.ConfigMapInterface
}

func (k *kubeProvider) GetEvaluableFeature(ctx context.Context, key string, namespace string) (feature.EvaluableFeature, error) {
	// TODO export this from the cli and import from here.
	cm, err := k.kubeClient.Get(ctx, fmt.Sprintf("lekko.%s", namespace), metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error fetching kubernetes namespace")
	}

	fBytes, ok := cm.BinaryData[key]
	if !ok {
		return nil, fmt.Errorf(fmt.Sprintf("could not find key: %s in namespace %s via k8s configmap", key, namespace))
	}
	// This is quite gross, we need to think of a better abstraction here... maybe a `ParseFeatureRaw` method
	// in encoding? Needs a bit more thought.
	evalF, err := encoding.ParseFeature("", feature.FeatureFile{}, &metadata.NamespaceConfigRepoMetadata{Version: "v1beta3"}, &kubeFileProvider{fBytes})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error deserializing feature for key %s, in namespace %s via k8s configmap", key, namespace))
	}
	return evalF, nil
}

func (k *kubeProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	evalF, err := k.GetEvaluableFeature(ctx, key, namespace)
	if err != nil {
		return false, err
	}
	resp, err := evalF.Evaluate(fromContext(ctx))
	if err != nil {
		return false, err
	}
	boolVal := new(wrapperspb.BoolValue)
	if !resp.MessageIs(boolVal) {
		return false, fmt.Errorf("invalid type in config map %T", resp)
	}
	if err := resp.UnmarshalTo(boolVal); err != nil {
		return false, err
	}
	return boolVal.Value, nil
}

func (k *kubeProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	return "", fmt.Errorf("unimplemented")
}
func (k *kubeProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result proto.Message) error {
	evalF, err := k.GetEvaluableFeature(ctx, key, namespace)
	if err != nil {
		return err
	}
	resp, err := evalF.Evaluate(fromContext(ctx))
	if err != nil {
		return err
	}
	if err := resp.UnmarshalTo(result); err != nil {
		return err
	}
	return nil
}
func (k *kubeProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	evalF, err := k.GetEvaluableFeature(ctx, key, namespace)
	if err != nil {
		return err
	}
	resp, err := evalF.Evaluate(fromContext(ctx))
	if err != nil {
		return err
	}
	val := &structpb.Value{}
	if !resp.MessageIs(val) {
		return fmt.Errorf("invalid type in config map %T", resp)
	}
	if err := resp.UnmarshalTo(val); err != nil {
		return fmt.Errorf("failed to unmarshal any to value: %w", err)
	}
	bytes, err := val.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal value into bytes: %w", err)
	}
	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to unmarshal json into go type %T", result))
	}
	return nil
}

type kubeFileProvider struct {
	contents []byte
}

func (k *kubeFileProvider) GetFileContents(_ context.Context, _ string) ([]byte, error) {
	return k.contents, nil
}

func (*kubeFileProvider) GetDirContents(ctx context.Context, path string) ([]fs.ProviderFile, error) {
	return nil, fmt.Errorf("unimplemented")
}
func (*kubeFileProvider) IsNotExist(err error) bool {
	return false
}
