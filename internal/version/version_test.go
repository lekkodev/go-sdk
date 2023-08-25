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

package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/semver"
)

func TestVersion(t *testing.T) {
	assert.True(t, semver.IsValid(Version))
	parts := strings.Split(SDKVersion, "-")
	assert.Len(t, parts, 2)
	assert.Equal(t, "go", parts[0])
	assert.True(t, semver.IsValid(parts[1]))
}
