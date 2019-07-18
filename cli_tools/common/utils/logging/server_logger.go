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
	"sync"
	"time"

	"github.com/google/uuid"
)

var(
	httpClient = &http.Client{}
	ServerUrl = ServerUrlProd // TODO
)

const(
	ServerUrlProd = "https://play.googleapis.com/log"
	ServerUrlTest = "http://localhost:27910/log"
)

// ServerLogger is responsible for logging to server
type ServerLogger struct {
	ServerUrl string
	Id        string
}

// ServerLoggerInterface is server logger abstraction
type ServerLoggerInterface interface {
	ServerLogSuccess()
	ServerLogFailure(reason string)
}

// NewServerLogger creates a new server logger
func NewServerLogger(serverUrl string) *ServerLogger {
	return &ServerLogger{
		ServerUrl: serverUrl,
		Id: uuid.New().String(),
	}
}

// LogStart logs a "start" info to server
func (l *ServerLogger) LogStart(params *ImportExportParamLog) {
	logEvent := &ImportExportLogEvent{
		Id:                   l.Id,
		Status:               "Start",
		ImportExportParamLog: params,
	}

	l.sendLogByHttp(logEvent)
}

// LogSuccess logs a "success" info to server
func (l *ServerLogger) LogSuccess(extraInfo *map[string]string) {
	logEvent := &ImportExportLogEvent{
		Id:                   l.Id,
		Status:               "Success",
		ExtraInfo:            extraInfo,
	}

	l.sendLogByHttp(logEvent)
}

// LogFailure logs a "failure" info to server
func (l *ServerLogger) LogFailure(reason string, extraInfo *map[string]string) {
	logEvent := &ImportExportLogEvent{
		Id:                   l.Id,
		Status:               "Failure",
		FailureReason:        reason,
		ExtraInfo:            extraInfo,
	}

	l.sendLogByHttp(logEvent)
}

// RunWithServerLogging runs the function with server logging
func RunWithServerLogging(params *ImportExportParamLog, function func()(*map[string]string, error)) error {
	l := NewServerLogger(ServerUrl)

	// Send log asynchronously. No need to interrupt the main flow when failed to send log, just
	// keep moving.
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.LogStart(params)
	} ()

	var extraInfo *map[string]string
	var err error
	if extraInfo, err = function(); err != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.LogFailure(err.Error(), extraInfo)
		} ()
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.LogSuccess(extraInfo)
		} ()
	}

	wg.Wait()
	return err
}

func (l *ServerLogger) sendLogByHttp(logEvent *ImportExportLogEvent) {
	logRequestJson, err := constructLogRequest(logEvent)
	fmt.Println(">>>", string(logRequestJson)) // TODO: remove
	if err != nil {
		fmt.Println("Failed to log to server: failed to prepare json log data.")
		return
	}
	resp, err := httpClient.Post(l.ServerUrl, "application/json", bytes.NewBuffer(logRequestJson))
	if err != nil {
		fmt.Println("Failed to log to server: ", err)
		return
	}

	// TODO: remove
	defer resp.Body.Close()
	respStr, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(">>>", string(respStr))
}

type LogRequest struct {
	ClientInfo ClientInfo `json:"client_info"`
	LogSourceName string `json:"log_source_name"`
	RequestTimeMs int64 `json:"request_time_ms"`
	LogEvent []LogEvent `json:"log_event"`
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

// ImportExportLogEvent is align with clearcut server side configuration.
type ImportExportLogEvent struct {
	// This id is a random guid for correlation among multiple log lines of a single call
	Id string `json:"id"`

	Status        string `json:"status"`
	FailureReason string `json:"failure_reason,omitempty"`
	ExtraInfo     *map[string]string `json:"extra_info,omitempty"`

	*ImportExportParamLog
}

// ImportExportParamLog contains the union of all APIs' param info.
type ImportExportParamLog struct {
	ClientId string `json:"client_id"`
}

func constructLogRequest(event *ImportExportLogEvent)([]byte, error) {
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
		LogEvent: []LogEvent{
			{
				EventTimeMs: now.Unix(),
				SequencePosition: 1,
				SourceExtensionJson: string(eventStr),
			},
		},
	}

	reqStr, err := json.Marshal(req)
	return reqStr, err
}