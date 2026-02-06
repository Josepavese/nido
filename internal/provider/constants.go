package provider

// Hardware Class Constants (PCI)
const (
	ClassDisplay      = "03"   // Display controller
	ClassMultimedia   = "04"   // Multimedia controller
	ClassNetwork      = "02"   // Network controller
	ClassSerialBus    = "0c"   // Serial bus controller
	ClassSerialBusUSB = "0c03" // USB controller
	ClassProcessing   = "12"   // Processing accelerators
	ClassBridge       = "06"   // Bridge device
	ClassStorage      = "01"   // Storage controller
)

// Human Readable Device Classes
const (
	DescDisplay    = "Display controller"
	DescNetwork    = "Network controller"
	DescMultimedia = "Multimedia controller"
	DescUSB        = "USB controller"
	DescSerial     = "Serial bus controller"
	DescProcessing = "Processing accelerator"
	DescUnknown    = "Unknown"
)
