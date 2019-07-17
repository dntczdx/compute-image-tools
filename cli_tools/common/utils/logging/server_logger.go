//  Copyright 2019 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package logging


// ServerLogger is responsible for logging to server
type ServerLogger struct {
	ServerUrl string
}

// ServerLoggerInterface is server logger abstraction
type ServerLoggerInterface interface {
	ServerLogSuccess()
	ServerLogFailure(reason string)
}

// NewServerLogger creates a new server logger
func NewServerLogger(serverUrl string) *ServerLogger {
	return &ServerLogger{ServerUrl: serverUrl}
}

// LogStart logs a "start" info to server
func (l *ServerLogger) LogStart() {
	// TODO
}

// LogSuccess logs a "success" info to server
func (l *ServerLogger) LogSuccess(params map[string]string, extraInfo map[string]string) {
	// TODO
}

// LogFailure logs a "failure" info to server
func (l *ServerLogger) LogFailure(reason string, params map[string]string, extraInfo map[string]string) {
	// TODO
}

func RunWithServerLogging(function func()(map[string]string, map[string]string, error)) error {
	sl := NewServerLogger("")
	sl.LogStart()
	if params, extraInfo, err := function(); err != nil {
		sl.LogFailure(err.Error(), params, extraInfo)
		return err
	} else {
		sl.LogSuccess(params, extraInfo)
		return nil
	}
}
