package testutil

import "github.com/google/uuid"

// Test UUIDs for consistent test data
var (
	TestChannelUUID1 = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	TestChannelUUID2 = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	TestDeviceUUID1  = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	TestDeviceUUID2  = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	TestAppletUUID1  = uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	TestAppletUUID2  = uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
)

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int
func IntPtr(i int) *int {
	return &i
}
