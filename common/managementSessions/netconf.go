package managementSessions

import (
	"OpenCNC/common/structures/credentials"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
	"golang.org/x/crypto/ssh"
)

var logger = slog.New(
	slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelWarn,
		},
	),
)

// CreateSession connects to a NETCONF server and returns the session.
func CreateSession(host, user, pass string) (*netconf.Session, error) {
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	address := fmt.Sprintf("%s:830", host)

	session, err := netconf.NewSessionFromSSHConfig(address, sshConfig, netconf.WithSessionLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	err = session.SendHello(&message.Hello{Capabilities: netconf.DefaultCapabilities})
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	return session, nil
}

// CreateSessionWithPassword connects using UsernamePassword credentials.
func CreateSessionWithPassword(host string, creds *credentials.UsernamePassword) (*netconf.Session, error) {
	return createSessionWithPasswordOnPort(host, 830, creds)
}

func createSessionWithPasswordOnPort(host string, port uint32, creds *credentials.UsernamePassword) (*netconf.Session, error) {
	if creds == nil {
		return nil, fmt.Errorf("username/password credentials are required")
	}

	sshConfig := &ssh.ClientConfig{
		User:            creds.GetUsername(),
		Auth:            []ssh.AuthMethod{ssh.Password(creds.GetPassword())},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	address := net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10))

	session, err := netconf.NewSessionFromSSHConfig(address, sshConfig, netconf.WithSessionLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	err = session.SendHello(&message.Hello{Capabilities: netconf.DefaultCapabilities})
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	return session, nil
}

// GetRunningConfig retrieves the <running> config using a <get-config> RPC.
func GetRunningConfig(session *netconf.Session) (string, error) {
	rpc := message.NewGetConfig(message.DatastoreRunning, "", "")
	reply, err := session.SyncRPC(rpc, 5)
	if err != nil {
		return "", fmt.Errorf("RPC failed: %w", err)
	}

	if reply == nil || reply.RawReply == "" {
		return "", fmt.Errorf("empty reply from device")
	}

	return reply.RawReply, nil
}

// EditConfig sends an <edit-config> RPC with the XML payload to <running>.
func EditConfig(session *netconf.Session, xmlData string) error {
	rpc := message.NewEditConfig(
		message.DatastoreRunning,
		message.DefaultOperationTypeMerge,
		xmlData,
	)
	reply, err := session.SyncRPC(rpc, 5)
	if err != nil {
		return fmt.Errorf("edit-config RPC failed: %w", err)
	}

	if reply == nil || reply.RawReply == "" {
		return fmt.Errorf("empty reply from edit-config")
	}

	if err := checkNetconfOKReply(reply.RawReply); err != nil {
		return fmt.Errorf("edit-config failed: %w", err)
	}

	fmt.Println("edit-config reply:")
	fmt.Println(reply.RawReply)

	return nil
}

// GetWithFilter retrieves operational state using NETCONF <get> and a subtree filter.
func GetWithFilter(session *netconf.Session, filter string) (string, error) {
	rpc := message.NewGet("subtree", filter)
	reply, err := session.SyncRPC(rpc, 5)
	if err != nil {
		return "", fmt.Errorf("get RPC failed: %w", err)
	}

	if reply == nil || reply.RawReply == "" {
		return "", fmt.Errorf("empty reply from get")
	}

	return reply.RawReply, nil
}

func checkNetconfOKReply(rawReply string) error {
	var rpcReply struct {
		OK     *struct{} `xml:"ok"`
		Errors []struct {
			Type     string `xml:"error-type"`
			Tag      string `xml:"error-tag"`
			Severity string `xml:"error-severity"`
			AppTag   string `xml:"error-app-tag"`
			Path     string `xml:"error-path"`
			Message  string `xml:"error-message"`
			Info     struct {
				XML string `xml:",innerxml"`
			} `xml:"error-info"`
		} `xml:"rpc-error"`
	}

	if err := xml.Unmarshal([]byte(rawReply), &rpcReply); err != nil {
		return fmt.Errorf("failed parsing NETCONF reply: %w", err)
	}
	if len(rpcReply.Errors) > 0 {
		details := make([]string, 0, len(rpcReply.Errors))
		for _, rpcError := range rpcReply.Errors {
			fields := []string{}
			for _, field := range []struct{ name, value string }{
				{"type", rpcError.Type}, {"tag", rpcError.Tag},
				{"severity", rpcError.Severity}, {"app-tag", rpcError.AppTag},
				{"path", rpcError.Path}, {"message", rpcError.Message},
				{"info", rpcError.Info.XML},
			} {
				if value := strings.TrimSpace(field.value); value != "" {
					fields = append(fields, fmt.Sprintf("%s=%q", field.name, value))
				}
			}
			details = append(details, "rpc-error: "+strings.Join(fields, ", "))
		}
		return fmt.Errorf("NETCONF %s", strings.Join(details, "; "))
	}

	if rpcReply.OK == nil {
		return fmt.Errorf("NETCONF reply does not contain <ok/>")
	}

	return nil
}
