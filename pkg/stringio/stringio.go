package stringio

import (
	"errors"
	"fmt"
	"syscall"
)

const buf_size = 4096

var OSError error = errors.New("I/O operation on closed file")

// A StringIO object is similar to a File object.
// It mimics all File I/O operations by implementing the
// common interfaces where File is implemented.
// The main difference is StringIO never read/write to filesystem.
// All operations are done in memory by accessing its underly buffer.
// The difference b/w bytes.Buffer is that StringIO supports
// Random access where Buffer does not.
// Most buffer operations are similar to bytes.Buffer.
// StringIO also does not support any non I/O operations such as
// Mkdir, Stats, Symlink, etc which does not have a real semantics for
// buffer manipulation.
// A StringIO object can not be reused once it is closed, just like
// the file object.
type StringIO struct {
	buf       []byte
	isclosed  bool
	pos, last int
	name      string
}

// Factory method served as the constructor.
func New() *StringIO {
	buf := make([]byte, buf_size)
	sio := new(StringIO)
	sio.buf = buf
	sio.isclosed = false
	sio.pos = 0
	sio.last = 0
	sio.name = fmt.Sprintf("StringIO <%p>", sio)
	return sio
}

func (s *StringIO) Len() int { return len(s.buf[s.pos:s.last]) }

// Query for stringio object's fd is an error.
func (s *StringIO) Fd() (fd int, err error) {
	return -1, errors.New("invalid")
}

func (s *StringIO) GoString() string { return s.name }

func (s *StringIO) Name() string { return s.name }

// Return the unread buffer.
func (s *StringIO) String() string {
	if s.isClosed() {
		return "<nil>"
	}
	return string(s.buf[s.pos:s.last])
}

// Return stored buffer until the last written position.
func (s *StringIO) GetValueString() string {
	if s.isClosed() {
		return "<nil>"
	}
	return string(s.buf[0:s.last])
}

// Return stored buffer as a byte array.
func (s *StringIO) GetValueBytes() []byte {
	if s.isClosed() {
		return s.buf[0:0]
	}
	return s.buf[0:s.last]
}

// Call Close will release the buffer/memory.
func (s *StringIO) Close() (err error) {
	s.Truncate(0)
	s.isclosed = true
	s.name = "StringIO <closed>"
	return
}

func (s *StringIO) Truncate(n int) {
	if s.isClosed() != true {
		if n == 0 {
			s.pos = 0
			s.last = 0
		}
		s.last = s.pos + n
		s.buf = s.buf[0:s.last]
	}
}

func (s *StringIO) Seek(offset int64, whence int) (ret int64, err error) {
	if s.isClosed() {
		return 0, OSError
	}
	pos, length := int64(s.pos), int64(len(s.buf))
	int64_O := int64(0)
	switch whence {
	case 0:
		ret = offset
	case 1:
		ret = offset + pos
	case 2:
		ret = offset + length
	default:
		return 0, errors.New("invalid")
	}
	if ret < int64_O {
		ret = int64_O
	}
	// StringIO currently does not support Seek beyond the
	// buf end, whereas posix does allow seek outside of
	// the file size, which will end up with a file hole.
	// However, StringIO does allow a byte hold within its
	// buffer size.
	if ret > length {
		ret = length
	}
	// Unfortunately, this will have to be a downcast.
	s.pos = int(ret)
	return
}

func (s *StringIO) Read(b []byte) (n int, err error) {
	if s.isClosed() {
		return 0, OSError
	}
	if s.pos >= len(s.buf) {
		return 0, errors.New("eof")
	}
	return s.readBytes(b)
}

func (s *StringIO) ReadAt(b []byte, offset int64) (n int, err error) {
	if s.isClosed() {
		return 0, OSError
	}
	s.setPos(offset)
	return s.readBytes(b)
}

// StringIO Write will always be success until memory is used up
// or system limit is reached.
func (s *StringIO) Write(b []byte) (n int, err error) {
	if s.isClosed() {
		return 0, OSError
	}
	return s.writeBytes(b)
}

func (s *StringIO) WriteAt(b []byte, offset int64) (n int, err error) {
	if s.isClosed() {
		return 0, OSError
	}
	s.setPos(offset)
	return s.writeBytes(b)
}

func (s *StringIO) WriteString(str string) (ret int, err error) {
	b := syscall.StringByteSlice(str)
	return s.Write(b[0 : len(b)-1])
}

// private methods
func (s *StringIO) readBytes(b []byte) (n int, err error) {
	if s.pos > s.last {
		return 0, nil
	}
	n = len(b)
	// Require more than what we have only get what we have.
	// In other words, empty bytes will not be sent out.
	if s.pos+n > s.last {
		n = s.last - s.pos
	}
	copy(b, s.buf[s.pos:s.pos+n])
	s.pos += n
	return
}

func (s *StringIO) writeBytes(b []byte) (n int, err error) {
	n = len(b)
	if n > s.length() {
		s.resize(n)
	}
	copy(s.buf[s.pos:s.pos+n], b)
	s.pos += n
	if s.pos > s.last {
		s.last = s.pos
	}
	return
}

func (s *StringIO) setPos(offset int64) {
	pos, int64_O, length := int64(s.pos), int64(0), int64(len(s.buf))
	pos = offset
	if offset < int64_O {
		pos = int64_O
	}
	if offset > length {
		pos = length
	}
	s.pos = int(pos)
}

func (s *StringIO) length() int { return len(s.buf) - s.pos }

func (s *StringIO) isClosed() bool { return s.isclosed == true }

// Stolen from bytes.Buffer (Use the same algorithm)
func (s *StringIO) resize(n int) {
	if len(s.buf)+n > cap(s.buf) {
		buf := make([]byte, 2*cap(s.buf)+n)
		copy(buf, s.buf[0:])
		s.buf = buf
	}
}
