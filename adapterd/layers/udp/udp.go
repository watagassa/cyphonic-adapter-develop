package udp

// UDP datagram length of each field.
const (
	SrcPortLength          = 2
	DstPortLength          = 2
	LengthLength           = 2
	ChecksumLength         = 2
	UDPDatagramTotalLength = SrcPortLength + DstPortLength + LengthLength + ChecksumLength
)
