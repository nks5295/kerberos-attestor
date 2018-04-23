package common

import (
	"fmt"

	gokrb_msgs "github.com/nks5295/gokrb5/messages"
)

const (
	SPIREServiceName = "DNS"
)

type KrbAttestedData struct {
	KrbAPReq gokrb_msgs.APReq
}

func AttestationStepError(step string, cause error) error {
	return fmt.Errorf("Attempted Kerberos attestation, but an error occured: %s: %s", step, cause)
}
