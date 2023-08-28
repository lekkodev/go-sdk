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
	_ "embed"
	"fmt"
	"strings"
)

var (
	//go:embed version.txt
	embedVersion string
	version      = readVersion()
	SDKVersion   = fmt.Sprintf("go-%s", version)
)

func readVersion() string {
	v := strings.TrimSpace(embedVersion)
	if len(v) == 0 {
		return "unknown"
	}
	if strings.HasPrefix(v, "v") {
		return v
	}
	return fmt.Sprintf("v%s", v)
}
