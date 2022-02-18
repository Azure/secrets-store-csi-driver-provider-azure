package utils

import "testing"

func TestRedactSecureString(t *testing.T) {
	tests := []struct {
		name            string
		sensitiveString string
		expected        string
	}{
		{
			name:            "should redact client id",
			sensitiveString: "aabc0000-a83v-9h4m-000j-2c0a66b0c1f9",
			expected:        "aabc##### REDACTED #####c1f9",
		},
		{
			name:            "should redact access token",
			sensitiveString: `eyJhbGciOiJSUzI1NiIsImtpZCI6InRhVDBxbzhQVEZ1ajB1S3BYUUxIclRsR01XakxjemJNOTlzWVMxSlNwbWcifQ.eyJhdWQiOlsiYXBpOi8vQXp1cmVBRGlUb2tlbkV4Y2hhbmdlIl0sImV4cCI6MTY0MzIzNDY0NywiaWF0IjoxNjQzMjMxMDQ3LCJpc3MiOiJodHRwczovL2t1YmVybmV0ZXMuZGVmYXVsdC5zdmMuY2x1c3Rlci5sb2NhbCIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoidGVzdC12MWFscGhhMSIsInBvZCI6eyJuYW1lIjoic2VjcmV0cy1zdG9yZS1pbmxpbmUtY3JkIiwidWlkIjoiYjBlYmZjMzUtZjEyNC00ZTEyLWI3N2UtYjM0MjM2N2IyMDNmIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiMjViNGY1NzgtM2U4MC00NTczLWJlOGQtZTdmNDA5ZDI0MmI2In19LCJuYmYiOjE2NDMyMzEwNDcsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDp0ZXN0LXYxYWxwaGExOmRlZmF1bHQifQ.ALE46aKmtTV7dsuFOwDZqvEjdHFUTNP-JVjMxexTemmPA78fmPTUZF0P6zANumA03fjX3L-MZNR3PxmEZgKA9qEGIDsljLsUWsVBEquowuBh8yoBYkGkMJmRfmbfS3y7_4Q7AU3D9Drw4iAHcn1GwedjOQC0i589y3dkNNqf8saqHfXkbSSLtSE0f2uzI-PjuTKvR1kuojEVNKlEcA4wsKfoiRpkua17sHkHU0q9zxCMDCr_1f8xbigRnRx0wscU3vy-8KhF3zQtpcWkk3r4C5YSXut9F3xjz5J9DUQn2vNMfZg4tOdcR-9Xv9fbY5iujiSlS58GEktSEa3SE9wrCw`,
			expected:        "eyJh##### REDACTED #####wrCw",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := RedactSecureString(test.sensitiveString)
			if actual != test.expected {
				t.Fatalf("expected: %s, got %s", test.expected, actual)
			}
		})
	}
}
