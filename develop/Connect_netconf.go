package main

// usage:
// go run Connect_netconf.go -s 192.168.4.64 -g >> output.xml
// go run Connect_netconf.go -s 192.168.4.64 -e -f config.xml
// go run Connect_netconf.go -s 192.168.4.64 -g -f filter.xml

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
	"os"

	"github.com/openshift-telco/go-netconf-client/netconf"
	"github.com/openshift-telco/go-netconf-client/netconf/message"
	"golang.org/x/crypto/ssh"
)

// createSession connects to a NETCONF server and returns the session.
func createSession(host, user, pass string) (*netconf.Session, error) {
	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	address := fmt.Sprintf("%s:830", host)

	session, err := netconf.NewSessionFromSSHConfig(address, sshConfig, netconf.WithSessionLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	err = session.SendHello(&message.Hello{
		Capabilities: netconf.DefaultCapabilities,
	})
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to send hello: %w", err)
	}

	return session, nil
}

// getRunningConfig retrieves the <running> config using a <get-config> RPC.
func getRunningConfig(session *netconf.Session) (string, error) {
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

// getRunningConfigWithFilter retrieves the <running> config using a <get-config> RPC with an XML filter.
func getRunningConfigWithFilter(session *netconf.Session, filter string) error {
	rpc := message.NewGetConfig(message.DatastoreRunning, "subtree", filter)
	reply, err := session.SyncRPC(rpc, 5)
	if err != nil {
		return fmt.Errorf("get-config RPC failed: %w", err)
	}

	fmt.Println("Filtered running config:")
	fmt.Println(reply.RawReply)

	return nil
}

// editConfig sends an <edit-config> RPC with the provided XML payload to the <running> datastore.
func editConfig(session *netconf.Session, xmlData string) error {
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

	fmt.Println("edit-config reply:")
	fmt.Println(reply.RawReply)

	return nil
}

// loadXMLFromFile reads an XML file from disk and returns it as a string.
func loadXMLFromFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read XML file: %w", err)
	}
	return string(data), nil
}

func main() {
	// Define command-line flags
	host := flag.String("s", "", "IP address of the switch (e.g. 192.168.0.1)")
	getConfig := flag.Bool("g", false, "Get the running config from the device")
	editConfigFlag := flag.Bool("e", false, "Edit the running config on the device")
	filePath := flag.String("f", "", "Path to XML config file (used for -e or -g with filter)")

	flag.Parse()

	if *host == "" {
		fmt.Println("Hostname is required. Use -s to specify it.")
		flag.Usage()
		os.Exit(1)
	}

	if *getConfig && *editConfigFlag {
		log.Fatal("Cannot use -g (get) and -e (edit) at the same time")
	}

	user := "root"
	pass := ""

	session, err := createSession(*host, user, pass)
	if err != nil {
		log.Fatalf("Error creating NETCONF session: %v", err)
	}
	defer session.Close()

	if *getConfig {
		if *filePath != "" {
			// Load filter from file
			filterXml, err := loadXMLFromFile(*filePath)
			if err != nil {
				log.Fatalf("Failed to load filter XML from file: %v", err)
			}

			err = getRunningConfigWithFilter(session, filterXml)
			if err != nil {
				log.Fatalf("Failed to get filtered config: %v", err)
			}
		} else {
			config, err := getRunningConfig(session)
			if err != nil {
				log.Fatalf("Failed to get full running config: %v", err)
			}
			fmt.Println("Full running config:")
			fmt.Println(config)
		}
	} else if *editConfigFlag {
		if *filePath == "" {
			log.Fatal("You must specify a config file path with -f when using -e")
		}

		xmlPayload, err := loadXMLFromFile(*filePath)
		if err != nil {
			log.Fatalf("Error reading XML file: %v", err)
		}

		err = editConfig(session, xmlPayload)
		if err != nil {
			log.Fatalf("Error executing edit-config: %v", err)
		}
		fmt.Println("edit-config completed.")
	} else {
		fmt.Println("You must specify either -g to get config or -e to edit config.")
		flag.Usage()
	}
}
