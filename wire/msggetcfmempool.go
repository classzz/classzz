// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

// MsgGetCFMempool is a request for a filter of the remote peer's mempool.
type MsgGetCFMempool struct {
	FilterType FilterType
}

// CzzDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgGetCFMempool) CzzDecode(r io.Reader, pver uint32, _ MessageEncoding) error {
	return readElement(r, &msg.FilterType)
}

// CzzEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgGetCFMempool) CzzEncode(w io.Writer, pver uint32, _ MessageEncoding) error {
	return writeElement(w, msg.FilterType)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgGetCFMempool) Command() string {
	return CmdGetCFMempool
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgGetCFMempool) MaxPayloadLength(pver uint32) uint32 {
	return 1
}

// NewMsgGetCFMempool returns a new bitcoin getcfmempool message that conforms
// to the Message interface using the passed parameters.
func NewMsgGetCFMempool(filterType FilterType) *MsgGetCFMempool {
	return &MsgGetCFMempool{FilterType: filterType}
}
