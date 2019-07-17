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

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var(
	httpClient = &http.Client{}
	ServerUrl = ServerUrlTest // TODO
)

const(
	ServerUrlProd = "https://play.googleapis.com/log"
	ServerUrlTest = "https://jmt17.google.com/log"
)

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
	logEvent := ImportExportLogEvent{
		Status: "Start",
	}

	l.sendLogByHttp(logEvent)
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
	sl := NewServerLogger(ServerUrl)
	sl.LogStart()
	if params, extraInfo, err := function(); err != nil {
		sl.LogFailure(err.Error(), params, extraInfo)
		return err
	} else {
		sl.LogSuccess(params, extraInfo)
		return nil
	}
}

func (l *ServerLogger) sendLogByHttp(logEvent ImportExportLogEvent) {
	logRequestJson, err := constructLogRequest(logEvent)
	if err != nil {
		fmt.Println("Failed to log to server: failed to prepare json log data.")
		return
	}
	resp, err := httpClient.Post(l.ServerUrl, "application/json", bytes.NewBuffer(logRequestJson))
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("Failed to log to server: ", err)
		return
	}

	// TODO: remove
	fmt.Println("Request:", string(logRequestJson))
	fmt.Println("Logging status:", resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	os.Exit(0)
}

type LogRequest struct {
	ClientInfo ClientInfo `json:"client_info"`
	LogSourceName string `json:"log_source_name"`
	RequestTimeMs int64 `json:"request_time_ms"`
	LogEvent LogEvent `json:"log_event"`
}

type ClientInfo struct {
	ClientType string `json:"client_type"`
	DesktopClientInfo map[string]string `json:"desktop_client_info"`
}

type LogEvent struct {
	EventTimeMs int64 `json:"event_time_ms"`
	SequencePosition int `json:"sequence_position"`
	SourceExtensionJson string `json:"source_extension_json"`
}

// ImportExportLogEvent is align with clearcut server side configuration. It is the union of all APIs' log info.
type ImportExportLogEvent struct {
	Status string `json:"status"`
}

func constructLogRequest(event ImportExportLogEvent)([]byte, error) {
	eventStr, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	req := LogRequest {
		ClientInfo: ClientInfo{
			ClientType: "DESKTOP",
			DesktopClientInfo: map[string]string{"os": "win32"}, // TODO
		},
		LogSourceName: "CONCORD", //TODO
		RequestTimeMs: now.Unix(),
		LogEvent: LogEvent{
			EventTimeMs: now.Unix(),
			SequencePosition: 1,
			SourceExtensionJson: string(eventStr), // TODO
		},
	}

	reqStr, err := json.Marshal(req)
	return reqStr, err
}