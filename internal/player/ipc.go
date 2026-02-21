package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/sirupsen/logrus"
)

// IPCClient maneja la comunicación con mpv via IPC (JSON over socket/pipe)
type IPCClient struct {
	conn      net.Conn
	log       *logrus.Logger
	requestID uint64
	mu        sync.Mutex
	responses map[uint64]chan *ipcResponse
	closed    bool
}

// ipcRequest es el formato de request que mpv espera
type ipcRequest struct {
	Command   []interface{} `json:"command"`
	RequestID uint64        `json:"request_id,omitempty"`
}

// ipcResponse es el formato de respuesta de mpv
type ipcResponse struct {
	RequestID uint64      `json:"request_id,omitempty"`
	Error     string      `json:"error,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// NewIPCClient crea un nuevo cliente IPC y se conecta al socket/pipe de mpv.
// Espera hasta 5 segundos para que mpv cree el socket.
func NewIPCClient(path string, log *logrus.Logger) (*IPCClient, error) {
	var conn net.Conn
	var err error

	// Reintentar conexión durante 5s (mpv puede tardar en crear el socket)
	for i := 0; i < 50; i++ {
		if runtime.GOOS == "windows" {
			// En Windows, usar named pipe via go-winio
			conn, err = winio.DialPipe(path, nil)
		} else {
			// En Unix, usar socket Unix
			conn, err = net.Dial("unix", path)
		}

		if err == nil {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		return nil, fmt.Errorf("timeout conectando a mpv IPC: %w", err)
	}

	client := &IPCClient{
		conn:      conn,
		log:       log,
		responses: make(map[uint64]chan *ipcResponse),
	}

	// Goroutine para leer respuestas de mpv
	go client.readLoop()

	log.Debug("Cliente IPC conectado")
	return client, nil
}

// readLoop lee respuestas JSON de mpv y las despacha a los canales correspondientes
func (c *IPCClient) readLoop() {
	scanner := bufio.NewScanner(c.conn)
	for scanner.Scan() {
		line := scanner.Text()
		c.log.Debugf("mpv IPC recv: %s", line)

		var resp ipcResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			c.log.Warnf("Error parseando respuesta IPC: %v", err)
			continue
		}

		c.mu.Lock()
		if ch, ok := c.responses[resp.RequestID]; ok {
			ch <- &resp
			delete(c.responses, resp.RequestID)
		}
		c.mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		c.log.Debugf("IPC read loop terminó: %v", err)
	}
}

// Command envía un comando a mpv y espera la respuesta
func (c *IPCClient) Command(name string, args ...interface{}) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("cliente IPC cerrado")
	}

	c.requestID++
	reqID := c.requestID
	respChan := make(chan *ipcResponse, 1)
	c.responses[reqID] = respChan
	c.mu.Unlock()

	// Construir comando
	cmdArgs := make([]interface{}, 0, len(args)+1)
	cmdArgs = append(cmdArgs, name)
	cmdArgs = append(cmdArgs, args...)

	req := ipcRequest{
		Command:   cmdArgs,
		RequestID: reqID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error serializando comando: %w", err)
	}

	c.log.Debugf("mpv IPC send: %s", string(data))

	// Enviar comando
	c.mu.Lock()
	_, err = c.conn.Write(append(data, '\n'))
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("error enviando comando: %w", err)
	}

	// Esperar respuesta con timeout
	select {
	case resp := <-respChan:
		if resp.Error != "" && resp.Error != "success" {
			return fmt.Errorf("mpv error: %s", resp.Error)
		}
		return nil
	case <-time.After(5 * time.Second):
		c.mu.Lock()
		delete(c.responses, reqID)
		c.mu.Unlock()
		return fmt.Errorf("timeout esperando respuesta de mpv")
	}
}

// GetProperty obtiene el valor de una propiedad de mpv
func (c *IPCClient) GetProperty(name string) (interface{}, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("cliente IPC cerrado")
	}

	c.requestID++
	reqID := c.requestID
	respChan := make(chan *ipcResponse, 1)
	c.responses[reqID] = respChan
	c.mu.Unlock()

	req := ipcRequest{
		Command:   []interface{}{"get_property", name},
		RequestID: reqID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error serializando request: %w", err)
	}

	c.log.Debugf("mpv IPC send: %s", string(data))

	c.mu.Lock()
	_, err = c.conn.Write(append(data, '\n'))
	c.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("error enviando request: %w", err)
	}

	select {
	case resp := <-respChan:
		if resp.Error != "" && resp.Error != "success" {
			return nil, fmt.Errorf("mpv error: %s", resp.Error)
		}
		return resp.Data, nil
	case <-time.After(5 * time.Second):
		c.mu.Lock()
		delete(c.responses, reqID)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout esperando respuesta de mpv")
	}
}

// SetProperty establece el valor de una propiedad de mpv
func (c *IPCClient) SetProperty(name string, value interface{}) error {
	return c.Command("set_property", name, value)
}

// Close cierra la conexión IPC
func (c *IPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.log.Debug("Cerrando cliente IPC")

	// Cerrar todos los canales de respuesta pendientes
	for _, ch := range c.responses {
		close(ch)
	}
	c.responses = make(map[uint64]chan *ipcResponse)

	return c.conn.Close()
}
