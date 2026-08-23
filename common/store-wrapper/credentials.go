package storewrapper

import (
	"context"
	"fmt"
	"strings"

	credentials "OpenCNC_config_service/common/structures/credentials"
	"OpenCNC_config_service/config_service/pkg/managementSessions"

	"google.golang.org/protobuf/proto"
)

func StoreCredentials(credentialID string, creds *credentials.ManagementCredentials) error {

	if strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("credential ID cannot be empty")
	}

	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}

	raw, err := proto.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	key := "credentials." + credentialID

	if err := SendToStore(raw, key); err != nil {
		return fmt.Errorf(
			"failed to store credentials %s: %w",
			credentialID,
			err,
		)
	}

	return nil
}

func DeleteCredentials(credentialID string) error {

	if strings.TrimSpace(credentialID) == "" {
		return fmt.Errorf("credential ID cannot be empty")
	}

	client, err := createEtcdClient()
	if err != nil {
		return fmt.Errorf(
			"failed to create etcd client: %w",
			err,
		)
	}
	defer client.Close()

	key := strings.ReplaceAll(
		"credentials."+credentialID,
		".",
		"/",
	)

	resp, err := client.Delete(
		context.Background(),
		key,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to delete credentials %s: %w",
			credentialID,
			err,
		)
	}

	if resp.Deleted == 0 {
		return fmt.Errorf(
			"credentials %s not found",
			credentialID,
		)
	}

	return nil
}

func GetCredentials(credentialID string) (*credentials.ManagementCredentials, error) {

	if strings.TrimSpace(credentialID) == "" {
		return nil, fmt.Errorf("credential ID cannot be empty")
	}

	key := "credentials." + credentialID

	raw, err := GetFromStore(key)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get credentials %s: %w",
			credentialID,
			err,
		)
	}

	creds := &credentials.ManagementCredentials{}

	if err := proto.Unmarshal(raw, creds); err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal credentials %s: %w",
			credentialID,
			err,
		)
	}

	return creds, nil
}

type CredentialMetadata struct {
	ID             string
	Authentication string
	Username       string
}

func GetCredentialMetadata(id string) (*CredentialMetadata, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("credential ID cannot be empty")
	}

	creds, err := GetCredentials(id)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get credential metadata %s: %w",
			id,
			err,
		)
	}

	metadata := &CredentialMetadata{
		ID: id,
	}

	switch auth := creds.Authentication.(type) {

	case *credentials.ManagementCredentials_UsernamePassword:
		metadata.Authentication = "USERNAME_PASSWORD"
		metadata.Username = auth.UsernamePassword.GetUsername()

	case *credentials.ManagementCredentials_SshPrivateKey:
		metadata.Authentication = "SSH_PRIVATE_KEY"
		metadata.Username = auth.SshPrivateKey.GetUsername()

	case *credentials.ManagementCredentials_TlsClientCertificate:
		metadata.Authentication = "TLS_CLIENT_CERTIFICATE"

	case *credentials.ManagementCredentials_SnmpV2CCommunity:
		metadata.Authentication = "SNMP_V2C_COMMUNITY"

	case *credentials.ManagementCredentials_SnmpV3:
		metadata.Authentication = "SNMP_V3"
		metadata.Username = auth.SnmpV3.GetUsername()

	default:
		metadata.Authentication = "UNKNOWN"
	}

	return metadata, nil
}

func GetCredentialsList() ([]managementSessions.NodeCredentials, error) {
	rawData, err := getFromStoreWithPrefix("credentials.")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get credentials from store: %w",
			err,
		)
	}

	result := make([]managementSessions.NodeCredentials, 0, len(rawData.Kvs))

	for _, kv := range rawData.Kvs {
		creds := &credentials.ManagementCredentials{}

		if err := proto.Unmarshal(kv.Value, creds); err != nil {
			return nil, fmt.Errorf(
				"failed to unmarshal credentials from key %s: %w",
				string(kv.Key),
				err,
			)
		}

		credentialID := strings.TrimPrefix(
			string(kv.Key),
			"credentials.",
		)

		result = append(result, managementSessions.NodeCredentials{
			ID:          credentialID,
			Credentials: creds,
		})
	}

	return result, nil
}
