package server

import (
	"net"
	"sync"
	"time"

	"github.com/iamduo/workq/int/client"
	"github.com/iamduo/workq/int/prot"
)

// Handler interface for a Command
// Handlers are responsible for executing a command
type Handler interface {
	Exec(cmd *prot.Cmd) ([]byte, error)
}

// Command Router takes in a command name and returns a handler
type Router interface {
	Handler(cmd string) Handler
}

// Command Router Implementation
type CmdRouter struct {
	Handlers       map[string]Handler
	UnknownHandler Handler
}

// Handler by command name
func (c *CmdRouter) Handler(cmd string) Handler {
	if h, ok := c.Handlers[cmd]; ok {
		return h
	}

	return c.UnknownHandler
}

// Workq Server listens on a TCP Address
// Requires a Command Router and a Protocol Implementation
type Server struct {
	Addr     string // Network Address to listen on
	Router   Router
	Prot     prot.Interface
	ln       net.Listener
	mu       sync.Mutex
	stop     chan struct{}
	stats    Stats
	statlock sync.RWMutex
}

// New returns a initialized, but unstarted Server
func New(addr string, router Router, protocol prot.Interface) *Server {
	return &Server{
		Addr:   addr,
		Router: router,
		Prot:   protocol,
		stop:   make(chan struct{}, 1),
	}
}

// ListenAndServe starts a Workq Server, listening on the specified TCP address
func (s *Server) ListenAndServe() error {
	var err error
	s.mu.Lock()
	s.ln, err = net.Listen("tcp", s.Addr)
	s.mu.Unlock()
	if err != nil {
		return err
	}

	s.statlock.Lock()
	s.stats.Started = time.Now().UTC()
	s.statlock.Unlock()

	for {
		select {
		case <-s.stop:
			return nil
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				break
			}

			go func() {
				s.clientLoop(client.New(conn.(*net.TCPConn), prot.MaxRead))
			}()
		}
	}
}

// Stop; listening while maintaining all active connections
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stop <- struct{}{}
	if s.ln != nil {
		return s.ln.Close()
	}

	return nil
}

// Stats returns stats for the server at the current time.
func (s *Server) Stats() Stats {
	s.statlock.RLock()
	defer s.statlock.RUnlock()
	return s.stats
}

func (s *Server) clientLoop(c *client.Client) {
	s.statlock.Lock()
	s.stats.ActiveClients++
	s.statlock.Unlock()

	defer func() {
		s.statlock.Lock()
		s.stats.ActiveClients--
		s.statlock.Unlock()
	}()

	rdr := c.Reader()
	wrt := c.Writer()
	cls := c.Closer()

	for {
		c.ResetLimit()
		cmd, err := s.Prot.ParseCmd(rdr)
		if err != nil {
			// Client Conn Error, Fail Fast
			if err == prot.ErrReadErr {
				cls.Close()
				return
			}

			err = s.Prot.SendErr(wrt, err.Error())
			if err != nil {
				cls.Close()
				return
			}

			continue
		}

		handler := s.Router.Handler(cmd.Name)
		reply, err := handler.Exec(cmd)
		switch {
		case err != nil:
			err = s.Prot.SendErr(wrt, err.Error())
			if err != nil {
				cls.Close()
				return
			}
		default:
			err = s.Prot.SendReply(wrt, reply)
			if err != nil {
				cls.Close()
				return
			}
		}
	}
}

// Stats Data
type Stats struct {
	// Number of active clients current connected.
	ActiveClients uint64

	// Started represents the time immediately after the server starts listening.
	Started time.Time
}
