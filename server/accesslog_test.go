// Copyright 2020 Google LLC
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

// Package protostruct supports operations on the protocol buffer Struct message.
package server

import (
	"testing"
	"time"

	// "github.com/gogo/status"
	"github.com/apigee/apigee-remote-service-golib/analytics"
	"github.com/apigee/apigee-remote-service-golib/auth"
	"github.com/golang/protobuf/ptypes"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	v2 "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v2"
	als "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
)

func TestHandleHTTPAccessLogs(t *testing.T) {

	now := time.Now()
	nowUnix := now.UnixNano() / 1000000
	nowProto, err := ptypes.TimestampProto(now)
	if err != nil {
		t.Fatal(err)
	}

	dur := 7 * time.Millisecond
	thenUnix := now.Add(dur).UnixNano() / 1000000
	durProto := ptypes.DurationProto(dur)

	headers := map[string]string{
		headerAPI:            "api",
		headerAPIProducts:    "product1,product2",
		headerAccessToken:    "token",
		headerApplication:    "app",
		headerClientID:       "clientID",
		headerDeveloperEmail: "email@google.com",
		headerEnvironment:    "env",
		headerOrganization:   "org",
		headerScope:          "scope1 scope2",
	}

	path := "path"
	uri := "path?x=foo"
	userAgent := "some agent"
	clientIP := "client ip"
	var entries []*v2.HTTPAccessLogEntry
	entries = append(entries, &v2.HTTPAccessLogEntry{
		CommonProperties: &v2.AccessLogCommon{
			StartTime:                   nowProto,
			TimeToLastRxByte:            durProto,
			TimeToFirstUpstreamTxByte:   durProto,
			TimeToLastUpstreamTxByte:    durProto,
			TimeToFirstUpstreamRxByte:   durProto,
			TimeToLastUpstreamRxByte:    durProto,
			TimeToFirstDownstreamTxByte: durProto,
			TimeToLastDownstreamTxByte:  durProto,
		},
		Request: &v2.HTTPRequestProperties{
			Path:           uri,
			RequestMethod:  core.RequestMethod_GET,
			UserAgent:      userAgent,
			ForwardedFor:   clientIP,
			RequestHeaders: headers,
		},
		Response: &v2.HTTPResponseProperties{
			ResponseCode: &wrappers.UInt32Value{
				Value: 200,
			},
		},
	})

	msg := &als.StreamAccessLogsMessage_HttpLogs{
		HttpLogs: &als.StreamAccessLogsMessage_HTTPAccessLogEntries{
			LogEntry: entries,
		},
	}

	testAnalyticsMan := &testAnalyticsMan{}
	server := AccessLogServer{
		handler: &Handler{
			orgName:      headers[headerOrganization],
			envName:      headers[headerEnvironment],
			analyticsMan: testAnalyticsMan,
		},
	}
	server.handleHTTPLogs(msg)

	recs := testAnalyticsMan.records
	if len(recs) != len(entries) {
		t.Errorf("got: %d, want: %d", len(recs), len(entries))
	}

	rec := recs[0]
	if rec.APIProxy != headers[headerAPI] {
		t.Errorf("got: %s, want: %s", rec.APIProxy, headers[headerAPI])
	}
	if rec.ClientIP != clientIP {
		t.Errorf("got: %s, want: %s", rec.ClientIP, clientIP)
	}
	if rec.ClientReceivedEndTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.ClientReceivedEndTimestamp, thenUnix)
	}
	if rec.ClientReceivedStartTimestamp != nowUnix {
		t.Errorf("got: %d, want: %d", rec.ClientReceivedStartTimestamp, nowUnix)
	}
	if rec.ClientSentEndTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.ClientSentEndTimestamp, thenUnix)
	}
	if rec.ClientSentStartTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.ClientSentStartTimestamp, thenUnix)
	}

	// the following are handled in golib by record.ensureFields()
	// so we're skipping validation of them...

	// rec.RecordType skipped
	// product := strings.Split(headers[headerAPIProducts], ",")[0]
	// if rec.APIProduct != product {
	// 	t.Errorf("got: %s, want: %s", rec.APIProduct, product)
	// }
	// rec.APIProxyRevision skipped
	// if rec.AccessToken != headers[headerAccessToken] {
	// 	t.Errorf("got: %s, want: %s", rec.AccessToken, headers[headerAccessToken])
	// }
	// if rec.ClientID != headers[headerClientID] {
	// 	t.Errorf("got: %s, want: %s", rec.ClientID, headers[headerClientID])
	// }
	// if rec.DeveloperApp != headers[headerApplication] {
	// 	t.Errorf("got: %s, want: %s", rec.DeveloperApp, headers[headerApplication])
	// }
	// if rec.DeveloperEmail != headers[headerDeveloperEmail] {
	// 	t.Errorf("got: %s, want: %s", rec.DeveloperEmail, headers[headerDeveloperEmail])
	// }
	// if rec.Environment != headers[headerEnvironment] {
	// 	t.Errorf("got: %s, want: %s", rec.Environment, headers[headerEnvironment])
	// }
	// if rec.GatewayFlowID != flowID {
	// 	t.Errorf("got: %s, want: %s", rec.GatewayFlowID, flowID)
	// }
	// rec.GatewaySource skipped
	// if rec.Organization != headers[headerOrganization] {
	// 	t.Errorf("got: %s, want: %s", rec.Organization, headers[headerOrganization])
	// }

	if rec.RequestPath != path {
		t.Errorf("got: %s, want: %s", rec.RequestPath, path)
	}
	if rec.RequestURI != uri {
		t.Errorf("got: %s, want: %s", rec.RequestURI, uri)
	}
	if rec.RequestVerb != core.RequestMethod_GET.String() {
		t.Errorf("got: %s, want: %s", core.RequestMethod_GET.String(), uri)
	}
	if rec.ResponseStatusCode != 200 {
		t.Errorf("got: %d, want: %d", rec.ResponseStatusCode, 200)
	}

	if rec.TargetReceivedEndTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.TargetReceivedEndTimestamp, thenUnix)
	}
	if rec.TargetReceivedStartTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.TargetReceivedStartTimestamp, thenUnix)
	}
	if rec.TargetSentEndTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.TargetSentEndTimestamp, thenUnix)
	}
	if rec.TargetSentStartTimestamp != thenUnix {
		t.Errorf("got: %d, want: %d", rec.TargetSentStartTimestamp, thenUnix)
	}
	if rec.UserAgent != userAgent {
		t.Errorf("got: %s, want: %s", rec.UserAgent, userAgent)
	}
}

func TestTimeToUnix(t *testing.T) {
	now := time.Now()
	want := now.UnixNano() / 1000000

	nowProto, err := ptypes.TimestampProto(now)
	if err != nil {
		t.Fatal(err)
	}
	got := pbTimestampToUnix(nowProto)
	if got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	got = pbTimestampToUnix(nil)
	if got != 0 {
		t.Errorf("got: %d, want: %d", got, 0)
	}
}

func TestAddDurationUnix(t *testing.T) {
	now := time.Now()
	duration := 6 * time.Minute
	want := now.Add(duration).UnixNano() / 1000000

	nowProto, err := ptypes.TimestampProto(now)
	if err != nil {
		t.Fatal(err)
	}
	durationProto := ptypes.DurationProto(duration)
	got := pbTimestampAddDurationUnix(nowProto, durationProto)

	if got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	got = pbTimestampAddDurationUnix(nil, durationProto)
	if got != 0 {
		t.Errorf("got: %d, want: %d", got, 0)
	}

	got = pbTimestampAddDurationUnix(nowProto, nil)
	want = now.UnixNano() / 1000000
	if got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
}

type testAnalyticsMan struct {
	analytics.Manager
	records []analytics.Record
}

func (a *testAnalyticsMan) Start() error {
	a.records = []analytics.Record{}
	return nil
}
func (a *testAnalyticsMan) Close() {}
func (a *testAnalyticsMan) SendRecords(ctx *auth.Context, records []analytics.Record) error {

	a.records = append(a.records, records...)
	return nil
}
