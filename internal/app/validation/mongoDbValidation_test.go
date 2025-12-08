package validation

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestValidator_NotNilObjectID(t *testing.T) {
	testCases := map[string]struct {
		input          primitive.ObjectID
		isInputInvalid bool
	}{
		"nil object id":  {primitive.NilObjectID, true},
		"non-nil object": {primitive.NewObjectID(), false},
	}

	for caseName, test := range testCases {
		t.Run(caseName, func(t *testing.T) {
			err := New().NotNilObjectID("id", test.input).Validate()

			if test.isInputInvalid && err == nil {
				t.Fatalf("expected validation error for %s, got nil", test.input.Hex())
			}
			if !test.isInputInvalid && err != nil {
				t.Fatalf("expected no error for %s, got %v", test.input.Hex(), err)
			}
		})
	}
}
