package models

import "testing"

func TestPayloadRecordTypes(t *testing.T) {
	tests := []struct {
		name     string
		payload  RecordPayload
		expected RecordType
	}{
		{"login payload", LoginPayload{Login: "user", Password: "pass"}, RecordTypeLogin},
		{"text payload", TextPayload{Content: "hello"}, RecordTypeText},
		{"binary payload", BinaryPayload{Data: []byte{1, 2, 3}}, RecordTypeBinary},
		{"card payload", CardPayload{Number: "4111", CVV: "123"}, RecordTypeCard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.payload.RecordType(); got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}
