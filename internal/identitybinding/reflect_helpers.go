package identitybinding

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// extractProxyTransport extracts the policy.Transporter that the SDK's
// internal proxy configuration set on a WorkloadIdentityCredential
// created with EnableAzureProxy: true.
//
// This avoids copying the SDK's internal proxy package by using
// reflect+unsafe to extract the transport the SDK built internally.
//
// Field chain (verified against azidentity v1.14.0-beta.2):
//
//	WorkloadIdentityCredential.cred (*ClientAssertionCredential)
//	  .client (*confidentialClient)
//	    .opts (confidentialClientOptions)
//	      .ClientOptions (azcore.ClientOptions)  [embedded]
//	        .Transport (policy.Transporter)
func extractProxyTransport(wic *azidentity.WorkloadIdentityCredential) (policy.Transporter, error) {
	v := reflect.ValueOf(wic).Elem()

	cred := v.FieldByName("cred")
	if !cred.IsValid() || cred.IsNil() {
		return nil, fmt.Errorf("field 'cred' not found or nil on WorkloadIdentityCredential")
	}

	client := cred.Elem().FieldByName("client")
	if !client.IsValid() || client.IsNil() {
		return nil, fmt.Errorf("field 'client' not found or nil on ClientAssertionCredential")
	}

	opts := client.Elem().FieldByName("opts")
	if !opts.IsValid() {
		return nil, fmt.Errorf("field 'opts' not found on confidentialClient")
	}

	co := opts.FieldByName("ClientOptions")
	if !co.IsValid() {
		return nil, fmt.Errorf("field 'ClientOptions' not found on confidentialClientOptions")
	}

	tf := co.FieldByName("Transport")
	if !tf.IsValid() {
		return nil, fmt.Errorf("field 'Transport' not found on ClientOptions")
	}

	if tf.IsNil() {
		return nil, fmt.Errorf("no transport configured on credential")
	}

	// Use reflect.NewAt+unsafe to bypass the unexported-field taint on reflect.Value.
	transport := reflect.NewAt(tf.Type(), unsafe.Pointer(tf.UnsafeAddr())).Elem().Interface().(policy.Transporter)
	return transport, nil
}
