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

package debug

import (
	"encoding/json"
	"log/slog"
	"os"
	"strings"
)

func LogDebug(msg string, args ...any) {
	if isDebugMode {
		slog.Info(msg, serializeArgs(args...)...)
	}
}

func LogInfo(msg string, args ...any) {
	slog.Info(msg, serializeArgs(args...)...)
}

func LogError(msg string, args ...any) {
	slog.Error(msg, serializeArgs(args...)...)
}

// Returns a masked version of the string, with the first showLen
// characters visible.
func Mask(s string, showLen int) string {
	i := showLen
	if i >= len(s) {
		return s
	}
	return s[:i] + strings.Repeat("*", len(s)-showLen)
}

func serializeArgs(args ...any) []any {
	serialized := make([]any, len(args))
	for i, arg := range args {
		if m, ok := arg.(map[string]interface{}); ok {
			if jb, err := json.Marshal(m); err == nil {
				serialized[i] = string(jb)
			} else {
				serialized[i] = arg
			}
		} else {
			serialized[i] = arg
		}
	}
	return serialized
}

// just a switch for now, later we can expose log level and ability to set custom logger
var isDebugMode = os.Getenv("LEKKO_DEBUG") == "true" || os.Getenv("LEKKO_DEBUG") == "1"

// Initialization message for package
func init() {
	if isDebugMode {
		LogDebug("LEKKO_DEBUG=true, running with additional logging")
	} else {
		LogInfo("Set LEKKO_DEBUG environment variable to \"true\" to enable debug evaluation logs")
	}
}
