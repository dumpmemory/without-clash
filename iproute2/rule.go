package iproute2

import (
	"errors"
	"unsafe"

	"github.com/mdlayher/netlink"

	"golang.org/x/sys/unix"
)

type Rule struct {
	Invert               *BooleanAttr
	Src                  *IPCIDRAttr
	Dst                  *IPCIDRAttr
	IifName              *StringAttr
	OifName              *StringAttr
	Priority             *Uint32Attr
	Table                *Uint32Attr
	Goto                 *Uint32Attr
	Mark                 *Uint32Attr
	Mask                 *Uint32Attr
	UidRange             *Uint32PairAttr
	SrcPortRange         *Uint16PairAttr
	DstPortRange         *Uint16PairAttr
	SuppressPrefixLength *Uint32Attr
}

func rule2Message(rule *Rule, create bool) ([]byte, error) {
	if rule.Src != nil && rule.Dst != nil && rule.Src.Value.Addr().Is6() != rule.Dst.Value.Addr().Is6() {
		return nil, errors.New("src ip & dst ip version not match")
	}

	routeMessage := &unix.RtMsg{
		Family: unix.AF_INET,
		Table:  unix.RT_TABLE_UNSPEC,
		Type:   unix.FR_ACT_UNSPEC,
	}
	if rule.Table != nil {
		routeMessage.Type = unix.FR_ACT_TO_TBL
	} else if rule.Goto != nil {
		routeMessage.Type = unix.FR_ACT_GOTO
	} else {
		if create {
			routeMessage.Type = unix.FR_ACT_NOP
		}
	}
	if rule.Src != nil {
		if rule.Src.Value.Addr().Is6() {
			routeMessage.Family = unix.AF_INET6
		}
		routeMessage.Src_len = uint8(rule.Src.Value.Bits())
	}
	if rule.Dst != nil {
		if rule.Dst.Value.Addr().Is6() {
			routeMessage.Family = unix.AF_INET6
		}
		routeMessage.Dst_len = uint8(rule.Dst.Value.Bits())
	}
	if rule.Invert != nil && rule.Invert.Value {
		routeMessage.Flags |= unix.FIB_RULE_INVERT
	}

	attributes := netlink.NewAttributeEncoder()
	if rule.Src != nil {
		attributes.Bytes(unix.FRA_SRC, rule.Src.Value.Addr().AsSlice())
	}
	if rule.Dst != nil {
		attributes.Bytes(unix.FRA_DST, rule.Dst.Value.Addr().AsSlice())
	}
	if rule.IifName != nil {
		attributes.String(unix.FRA_IIFNAME, rule.IifName.Value)
	}
	if rule.OifName != nil {
		attributes.String(unix.FRA_OIFNAME, rule.OifName.Value)
	}
	if rule.Mark != nil {
		attributes.Uint32(unix.FRA_FWMARK, rule.Mark.Value)
	}
	if rule.Mask != nil {
		attributes.Uint32(unix.FRA_FWMASK, rule.Mark.Value)
	}
	if rule.Priority != nil {
		attributes.Uint32(unix.FRA_PRIORITY, rule.Priority.Value)
	}
	if rule.Table != nil {
		attributes.Uint32(unix.FRA_TABLE, rule.Table.Value)
	}
	if rule.Goto != nil {
		attributes.Uint32(unix.FRA_GOTO, rule.Goto.Value)
	}
	if rule.UidRange != nil {
		attributes.Bytes(unix.FRA_UID_RANGE, (*(*[8]byte)(unsafe.Pointer(rule.UidRange)))[:])
	}
	if rule.SrcPortRange != nil {
		attributes.Bytes(unix.FRA_SPORT_RANGE, (*(*[4]byte)(unsafe.Pointer(rule.SrcPortRange)))[:])
	}
	if rule.DstPortRange != nil {
		attributes.Bytes(unix.FRA_DPORT_RANGE, (*(*[4]byte)(unsafe.Pointer(rule.DstPortRange)))[:])
	}
	if rule.SuppressPrefixLength != nil {
		attributes.Uint32(unix.FRA_SUPPRESS_PREFIXLEN, rule.SuppressPrefixLength.Value)
	}

	attributesBlock, err := attributes.Encode()
	if err != nil {
		return nil, err
	}

	result := make([]byte, unix.SizeofRtMsg+len(attributesBlock))

	copy(result, (*(*[unix.SizeofRtMsg]byte)(unsafe.Pointer(routeMessage)))[:])
	copy(result[unix.SizeofRtMsg:], attributesBlock)

	return result, nil
}

func RuleAdd(rule *Rule) error {
	data, err := rule2Message(rule, true)
	if err != nil {
		return err
	}

	request := netlink.Message{
		Header: netlink.Header{
			Type:  unix.RTM_NEWRULE,
			Flags: netlink.Request | netlink.Create | netlink.Excl | netlink.Acknowledge,
		},
		Data: data,
	}

	conn, err := netlink.Dial(unix.NETLINK_ROUTE, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Execute(request)
	return err
}

func RuleDel(rule *Rule) error {
	data, err := rule2Message(rule, false)
	if err != nil {
		return err
	}

	request := netlink.Message{
		Header: netlink.Header{
			Type:  unix.RTM_DELRULE,
			Flags: netlink.Request | netlink.Acknowledge,
		},
		Data: data,
	}

	conn, err := netlink.Dial(unix.NETLINK_ROUTE, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Execute(request)
	return err
}
