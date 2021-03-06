// Network communication layer, responsible for the bottom layer of network communication,
// mainly including tcp && udp two protocol implementation
package transport

import (
	"context"
	"encoding/binary"
	"github.com/lubanproj/gorpc/codec"
	"github.com/lubanproj/gorpc/codes"
	"io"
	"net"
)

const DefaultPayloadLength = 1024
const MaxPayloadLength = 4 * 1024 * 1024

type ServerTransport interface {
	// monitoring and processing of requests
	ListenAndServe(context.Context, ...ServerTransportOption) error
}

type ClientTransport interface {
	// send requests
	Send(context.Context, []byte, ...ClientTransportOption) ([]byte, error)
}

type Framer interface {
	// read a full frame
	ReadFrame(net.Conn) ([]byte, error)
}

type framer struct {
	buffer []byte
	counter int  // to prevent the dead loop
}

func NewFramer() Framer {
	return &framer {
		buffer : make([]byte, DefaultPayloadLength),
	}
}

func (f *framer) Resize() {
	f.buffer = make([]byte, len(f.buffer) * 2)
}


func (f *framer) ReadFrame(conn net.Conn) ([]byte, error) {

	frameHeader := make([]byte, codec.FrameHeadLen)
	if num, err := io.ReadFull(conn, frameHeader); num != codec.FrameHeadLen || err != nil {
		return nil, err
	}

	// validate magic
	if magic := uint8(frameHeader[0]); magic != codec.Magic {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "invalid magic...")
	}

	length := binary.BigEndian.Uint32(frameHeader[7:11])

	if length > MaxPayloadLength {
		return nil, codes.NewFrameworkError(codes.ClientMsgErrorCode, "payload too large...")
	}

	for uint32(len(f.buffer)) < length && f.counter <= 12 {
		f.buffer = make([]byte, len(f.buffer) * 2)
		f.counter++
	}

	if num, err := io.ReadFull(conn, f.buffer[:length]); uint32(num) != length || err != nil {
		return nil, err
	}

	return append(frameHeader, f.buffer[:length] ...), nil
}


