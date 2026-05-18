package hardware

import (
	"context"
	"fmt"
)

// HardwareStub is a mock implementation of hardware interactions
type HardwareStub struct{}

func NewHardwareStub() *HardwareStub { return &HardwareStub{} }

// GetHWID returns a mocked hardware ID
func (h *HardwareStub) GetHWID(ctx context.Context) (string, error) {
	return "MOCK-HWID-1234", nil
}

// ValidateHMACSignature validates HMAC signature (mocked)
func (h *HardwareStub) ValidateHMACSignature(ctx context.Context, payload []byte, signature string) (bool, error) {
	// In production, implement real HMAC check
	return true, nil
}

// KickDrawer simulates opening the cash drawer
func (h *HardwareStub) KickDrawer(ctx context.Context) error {
	fmt.Println("[hardware] KickDrawer called")
	return nil
}

// PrintTicket simulates ESC/POS printing
func (h *HardwareStub) PrintTicket(ctx context.Context, data []byte) error {
	fmt.Println("[hardware] PrintTicket called: ", string(data))
	return nil
}
