package managementSessions

import (
	commonmgmt "OpenCNC_config_service/common/managementSessions"
	"OpenCNC_config_service/common/structures/credentials"

	"github.com/openshift-telco/go-netconf-client/netconf"
)

func CreateSession(host, user, pass string) (*netconf.Session, error) {
	return commonmgmt.CreateSession(host, user, pass)
}

func CreateSessionWithPassword(host string, creds *credentials.UsernamePassword) (*netconf.Session, error) {
	return commonmgmt.CreateSessionWithPassword(host, creds)
}

func GetRunningConfig(session *netconf.Session) (string, error) {
	return commonmgmt.GetRunningConfig(session)
}

func EditConfig(session *netconf.Session, xmlData string) error {
	return commonmgmt.EditConfig(session, xmlData)
}
