package common

import (
	"fmt"

	"github.com/nks5295/gokrb5/messages"
)

type KrbAttestedData struct {
	KrbApReq messages.APReq
}

func AttestationStepError(step string, cause error) error {
	return fmt.Errorf("Attempted Kerberos attestation, but an error occured: %s: %s", step, cause)
}
