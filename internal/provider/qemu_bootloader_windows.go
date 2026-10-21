package provider

// The Windows backend has no discoverable control endpoint for this optional
// boot-menu optimization. VM start does not claim that it ran.
func (*QemuProvider) skipBootloader(string) {}
