package managementSessions

import (
	commonmgmt "OpenCNC/common/managementSessions"
	"OpenCNC/common/structures/credentials"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

func CreateSession(host, user, pass string) (*netconf.Session, error) {
	return commonmgmt.CreateSession(host, user, pass)
}

func GetRunningConfig(session *netconf.Session) (string, error) {
	return commonmgmt.GetRunningConfig(session)
}

func EditConfig(session *netconf.Session, xmlData string) error {
	return commonmgmt.EditConfig(session, xmlData)
}

func GetWithFilter(session *netconf.Session, filter string) (string, error) {
	return commonmgmt.GetWithFilter(session, filter)
}

func CreateSessionWithPassword(host string, creds *credentials.UsernamePassword) (*netconf.Session, error) {
	return commonmgmt.CreateSessionWithPassword(host, creds)
}
