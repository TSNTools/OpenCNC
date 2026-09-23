package managementSessions

import (
	"strings"
	"testing"
)

func TestCheckNetconfOKReply(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		want        []string
	}{
		{"success", `<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><ok/></rpc-reply>`, nil},
		{"prefixed success", `<nc:rpc-reply xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"><nc:ok/></nc:rpc-reply>`, nil},
		{"device errors", `<nc:rpc-reply xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"><nc:rpc-error><nc:error-type>application</nc:error-type><nc:error-tag>invalid-value</nc:error-tag><nc:error-severity>error</nc:error-severity><nc:error-app-tag>bad-gate</nc:error-app-tag><nc:error-path>/interfaces/interface</nc:error-path><nc:error-message xml:lang="en">Invalid gate &amp; interval</nc:error-message><nc:error-info><nc:bad-element>interval</nc:bad-element></nc:error-info></nc:rpc-error><nc:rpc-error><nc:error-tag>operation-failed</nc:error-tag></nc:rpc-error></nc:rpc-reply>`, []string{`type="application"`, `tag="invalid-value"`, `severity="error"`, `app-tag="bad-gate"`, `path="/interfaces/interface"`, `message="Invalid gate & interval"`, `<nc:bad-element>interval</nc:bad-element>`, `tag="operation-failed"`}},
		{"error overrides ok", `<rpc-reply><ok/><rpc-error><error-tag>operation-failed</error-tag></rpc-error></rpc-reply>`, []string{"operation-failed"}},
		{"missing ok", `<rpc-reply/>`, []string{"does not contain <ok/>"}},
		{"malformed", `<rpc-reply>`, []string{"failed parsing NETCONF reply"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkNetconfOKReply(tc.reply)
			if tc.want == nil {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err, want)
				}
			}
		})
	}
}
