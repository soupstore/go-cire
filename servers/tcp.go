package servers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"github.com/google/uuid"
	"github.com/soupstoregames/go-core/logging"
	"net"
	"strings"
)

type Connection interface {
	WriteMessage(p []byte) (err error)
	BufferUpdate(s []byte)
	Flush(tick uint32)
	Logger() *logging.ConnectionLogger
}

type Server struct {
	Connections chan *TCPConnection

	listener net.Listener
	addr     string
	stopping bool
}

func NewTCPServer(addr string) *Server {
	return &Server{
		addr:        addr,
		Connections: make(chan *TCPConnection),
	}
}

func (t *Server) Start() error {
	var err error

	if t.listener, err = net.Listen("tcp", t.addr); err != nil {
		return err
	}

	logging.Info("TCP Server listening on " + t.addr)

	for {
		if t.stopping {
			break
		}
		// Listen for an incoming connection.
		conn, err := t.listener.Accept()
		if err != nil {
			// net.errClosing is not exported so this
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			logging.Error(err.Error())
		}

		if t.stopping {
			conn.Close()
			break
		}

		logging.Debug("Client connected: " + conn.RemoteAddr().String())

		t.Connections <- NewTCPConnection(conn)
	}

	return nil
}

func (t *Server) Stop() {
	logging.Info("Stopping TCP Server")
	t.stopping = true
	close(t.Connections)
}

type TCPConnection struct {
	*logging.ConnectionLogger
	Closed bool

	conn           net.Conn
	reader         *bufio.Reader
	id             string
	closeFunctions []func()
	updates        bytes.Buffer
}

func NewTCPConnection(c net.Conn) *TCPConnection {
	id := uuid.New().String()
	conn := &TCPConnection{
		ConnectionLogger: logging.BuildConnectionLogger(id),
		conn:             c,
		reader:           bufio.NewReader(c),
		id:               id,
	}

	return conn
}

func (c *TCPConnection) ID() string {
	return c.id
}

func (c *TCPConnection) Close() error {
	if c.Closed {
		return nil
	}

	c.Info("Closing connection")
	c.Closed = true
	for _, f := range c.closeFunctions {
		f()
	}
	return c.conn.Close()
}

func (c *TCPConnection) OnClose(f func()) {
	c.closeFunctions = append(c.closeFunctions, f)
}

func (c *TCPConnection) WriteMessage(p []byte) error {
	_, err := c.conn.Write(p)
	return err
}

func (c *TCPConnection) ReadMessage() ([]byte, error) {
	length := make([]byte, 2)
	if _, err := c.reader.Read(length); err != nil {
		if c.Closed {
			return []byte{}, nil
		}
		return []byte{}, err
	}

	body := make([]byte, binary.LittleEndian.Uint16(length))
	if _, err := c.reader.Read(body); err != nil {
		return []byte{}, err
	}

	return body, nil
}

func (c *TCPConnection) BufferUpdate(s []byte) {
	c.updates.Write(s)
}

func (c *TCPConnection) Flush(tick uint32) {
	if c.updates.Len() == 0 {
		return
	}

	b := make([]byte, 4, 1024)

	binary.LittleEndian.PutUint32(b[:], tick)
	b = append(b, c.updates.Bytes()...)
	b = append(b, 0)

	if err := c.WriteMessage(b); err != nil {
		if c.Closed {
			return
		}
		c.ConnectionLogger.WithError(err).Error("Failed to write updates")
		c.Close()
	}

	c.updates.Reset()
}

func (c *TCPConnection) Logger() *logging.ConnectionLogger {
	return c.ConnectionLogger
}
