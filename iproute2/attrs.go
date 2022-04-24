package iproute2

import (
	"net/netip"
)

type BooleanAttr struct {
	Value bool
}

type ByteAttr struct {
	Value byte
}

type Uint32Attr struct {
	Value uint32
}

type StringAttr struct {
	Value string
}

type Uint16PairAttr struct {
	First  uint16
	Second uint16
}

type Uint32PairAttr struct {
	First  uint32
	Second uint32
}

type IPCIDRAttr struct {
	Value netip.Prefix
}
