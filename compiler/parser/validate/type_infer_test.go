package validate

import (
	"testing"

	"github.com/MBlore/AuAu/ast"
)

func TestLiteralFitsType(t *testing.T) {
	type testCase struct {
		name          string
		literal       string
		base          int
		negative      bool
		targetType    *ast.TypeRef
		expectError   bool
		expectedError string
	}

	testCases := []testCase{
		{
			name:        "valid int literal fits int32",
			literal:     "123",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt32},
			expectError: false,
		},
		{
			name:        "valid int literal fits int8 max",
			literal:     "127",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt8},
			expectError: false,
		},
		{
			name:        "valid negative int literal fits int8 min",
			literal:     "128",
			base:        10,
			negative:    true,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt8},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds int8",
			literal:       "128",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt8},
			expectError:   true,
			expectedError: "integer literal 128 out of range for int8",
		},
		{
			name:          "invalid negative int literal exceeds int8",
			literal:       "129",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt8},
			expectError:   true,
			expectedError: "integer literal -129 out of range for int8",
		},
		{
			name:        "valid int literal fits int16 max",
			literal:     "32767",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt16},
			expectError: false,
		},
		{
			name:        "valid negative int literal fits int16 min",
			literal:     "32768",
			base:        10,
			negative:    true,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt16},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds int16",
			literal:       "32768",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt16},
			expectError:   true,
			expectedError: "integer literal 32768 out of range for int16",
		},
		{
			name:          "invalid negative int literal exceeds int16",
			literal:       "32769",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt16},
			expectError:   true,
			expectedError: "integer literal -32769 out of range for int16",
		},
		{
			name:        "valid negative int literal fits int32",
			literal:     "123",
			base:        10,
			negative:    true,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt32},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds int32",
			literal:       "2147483648",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt32},
			expectError:   true,
			expectedError: "integer literal 2147483648 out of range for int32",
		},
		{
			name:          "invalid negative int literal exceeds int32",
			literal:       "2147483649",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt32},
			expectError:   true,
			expectedError: "integer literal -2147483649 out of range for int32",
		},
		{
			name:        "valid int literal fits int64 max",
			literal:     "9223372036854775807",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt64},
			expectError: false,
		},
		{
			name:        "valid negative int literal fits int64 min",
			literal:     "9223372036854775808",
			base:        10,
			negative:    true,
			targetType:  &ast.TypeRef{Kind: ast.TypeInt64},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds int64",
			literal:       "9223372036854775808",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt64},
			expectError:   true,
			expectedError: "integer literal 9223372036854775808 out of range for int",
		},
		{
			name:          "invalid negative int literal exceeds int64",
			literal:       "9223372036854775809",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeInt64},
			expectError:   true,
			expectedError: "integer literal -9223372036854775809 out of range for int",
		},
		{
			name:        "valid int literal fits uint8",
			literal:     "255",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeUInt8},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds uint8",
			literal:       "256",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt8},
			expectError:   true,
			expectedError: "integer literal 256 out of range for uint8",
		},
		{
			name:          "invalid negative int literal for uint8",
			literal:       "1",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt8},
			expectError:   true,
			expectedError: "cannot assign negative integer literal -1 to unsigned type uint8",
		},
		{
			name:        "valid int literal fits uint16",
			literal:     "65535",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeUInt16},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds uint16",
			literal:       "65536",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt16},
			expectError:   true,
			expectedError: "integer literal 65536 out of range for uint16",
		},
		{
			name:          "invalid negative int literal for uint16",
			literal:       "1",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt16},
			expectError:   true,
			expectedError: "cannot assign negative integer literal -1 to unsigned type uint16",
		},
		{
			name:        "valid int literal fits uint32",
			literal:     "4294967295",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeUInt32},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds uint32",
			literal:       "4294967296",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt32},
			expectError:   true,
			expectedError: "integer literal 4294967296 out of range for uint32",
		},
		{
			name:          "invalid negative int literal for uint32",
			literal:       "1",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt32},
			expectError:   true,
			expectedError: "cannot assign negative integer literal -1 to unsigned type uint32",
		},
		{
			name:        "valid int literal fits uint64",
			literal:     "18446744073709551615",
			base:        10,
			negative:    false,
			targetType:  &ast.TypeRef{Kind: ast.TypeUInt64},
			expectError: false,
		},
		{
			name:          "invalid int literal exceeds uint64",
			literal:       "18446744073709551616",
			base:          10,
			negative:      false,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt64},
			expectError:   true,
			expectedError: "invalid integer literal \"18446744073709551616\"",
		},
		{
			name:          "invalid negative int literal for uint64",
			literal:       "1",
			base:          10,
			negative:      true,
			targetType:    &ast.TypeRef{Kind: ast.TypeUInt64},
			expectError:   true,
			expectedError: "cannot assign negative integer literal -1 to unsigned type uint64",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lit := &ast.IntLiteralExpr{Literal: tc.literal, Base: tc.base}
			err := literalFitsType(lit, tc.negative, tc.targetType)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected error %q but got %q", tc.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
