package syndicate

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gholt/ring"
	cc "github.com/pandemicsyn/cmdctrl/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	_FH_STOP_NODE_TIMEOUT = 60
)

func ParseManagedNodeAddress(addr string, port int) (string, error) {
	if addr == "" {
		return "", fmt.Errorf("address missing")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, port), nil
}

func bootstrapManagedNodes(ring ring.Ring, ccport int) map[uint64]*ManagedNode {
	nodes := ring.Nodes()
	m := make(map[uint64]*ManagedNode, len(nodes))
	for _, node := range nodes {
		addr, err := ParseManagedNodeAddress(node.Address(0), ccport)
		if err != nil {
			log.Printf("Error bootstrapping node %d: unable to split address %s: %v", node.ID(), node.Address(0), err)
			log.Println("Node NOT a managed node!")
			continue
		}
		m[node.ID()], err = NewManagedNode(addr)
		if err != nil {
			log.Printf("Error bootstrapping node %d: %v", node.ID(), err)
		} else {
			log.Println("Added", node.ID(), "as managed node")
		}
	}
	return m
}

type ManagedNode struct {
	sync.RWMutex
	failcount   int64
	ringversion int64
	active      bool
	conn        *grpc.ClientConn
	client      cc.CmdCtrlClient
	address     string
}

func NewManagedNode(address string) (*ManagedNode, error) {
	var err error
	if address == "" {
		return &ManagedNode{}, fmt.Errorf("Invalid Address supplied")
	}
	var opts []grpc.DialOption
	var creds credentials.TransportAuthenticator
	creds = credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	opts = append(opts, grpc.WithTransportCredentials(creds))
	s := &ManagedNode{}
	s.address = address
	s.conn, err = grpc.Dial(s.address, opts...)
	if err != nil {
		return &ManagedNode{}, fmt.Errorf("Failed to dial cmdctrl server for node %s: %v", s.address, err)
	}
	s.client = cc.NewCmdCtrlClient(s.conn)
	s.active = false
	s.failcount = 0
	return s, nil
}

// Take direct from grpc.Conn.WaitForStateChange:
// WaitForStateChange blocks until the state changes to something other than the sourceState
// or timeout fires. The grpc instance returns error if timeout fires or new ConnectivityState otherwise.
// Our instance returns if timeout fires or state changes OR returns state is Shutdown if n.conn is nil!
// I assume we'll wanna use this do things like update synd state when a node comes online after a failure
// or something.
func (n *ManagedNode) ConnWaitForStateChange(ctx context.Context, timeout time.Duration, sourceState grpc.ConnectivityState) (grpc.ConnectivityState, error) {
	n.Lock()
	defer n.Unlock()
	if n.conn != nil {
		return n.conn.WaitForStateChange(ctx, sourceState)
	}
	return grpc.Shutdown, nil
}

// ConnState returns the state of the underlying grpc connection.
// See https://godoc.org/google.golang.org/grpc#ConnectivityState for possible states.
// Returns -1 if n.conn is nil
func (n *ManagedNode) ConnState() (grpc.ConnectivityState, error) {
	n.RLock()
	defer n.RUnlock()
	if n.conn != nil {
		return n.conn.State()
	}
	return -1, nil
}

// Connect sets up a grpc connection for the node.
// Note that this will overwrite an existing conn.
func (n *ManagedNode) Connect() error {
	n.Lock()
	defer n.Unlock()
	var opts []grpc.DialOption
	var creds credentials.TransportAuthenticator
	var err error
	creds = credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	opts = append(opts, grpc.WithTransportCredentials(creds))
	n.conn, err = grpc.Dial(n.address, opts...)
	if err != nil {
		return fmt.Errorf("Failed to dial ring server for config: %v", err)
	}
	n.client = cc.NewCmdCtrlClient(n.conn)
	return nil
}

// Disconnect lets you disconnect a managed nodes grpc conn.
func (n *ManagedNode) Disconnect() error {
	n.Lock()
	defer n.Unlock()
	return n.conn.Close()
}

// Ping verifies a node as actually still alive.
func (n *ManagedNode) Ping() (bool, string, error) {
	n.RLock()
	defer n.RUnlock()
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	status, err := n.client.HealthCheck(ctx, &cc.EmptyMsg{})
	if err != nil {
		return false, "", err
	}
	return status.Status, status.Msg, err
}

// Stop a remote node
func (n *ManagedNode) Stop() error {
	n.Lock()
	defer n.Unlock()
	ctx, _ := context.WithTimeout(context.Background(), _FH_STOP_NODE_TIMEOUT*time.Second)
	status, err := n.client.Stop(ctx, &cc.EmptyMsg{})
	if err != nil {
		return err
	}
	n.active = status.Status
	return nil
}

// RingUpdate lets you push a ring update to a remote node
func (n *ManagedNode) RingUpdate(r *[]byte, version int64) (bool, error) {
	n.Lock()
	defer n.Unlock()
	if n.ringversion == version {
		return false, nil
	}
	connstate, err := n.conn.State()
	if err != nil {
		// TODO: reconnect
		return false, fmt.Errorf("Ring update of %s failed. grpc.conn err: %v", n.address, err)
	}
	if connstate != grpc.Ready {
		// TODO: reconnect
		return false, fmt.Errorf("Ring update of %s failed. grpc.conn not ready: %s", n.address, connstate.String())
	}
	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)
	ru := &cc.Ring{
		Ring:    *r,
		Version: version,
	}
	status, err := n.client.RingUpdate(ctx, ru)
	if err != nil {
		if status != nil {
			if status.Newversion == version {
				return true, err
			}
		}
		return false, err
	}
	n.ringversion = status.Newversion
	if n.ringversion != ru.Version {
		return false, fmt.Errorf("Ring update seems to have failed. Expected: %d, but remote host reports: %d\n", ru.Version, status.Newversion)
	}
	return true, nil
}

type changeMsg struct {
	rb *[]byte
	v  int64
}

// NotifyNodes is called when a ring change occur's and just
// drops a change message on the changeChan for the RingChangeManager.
func (s *Server) NotifyNodes() {
	s.RLock()
	m := &changeMsg{
		rb: s.rb,
		v:  s.r.Version(),
	}
	s.RUnlock()
	s.changeChan <- m
}

func (s *Server) RingChangeManager() {
	for msg := range s.changeChan {
		s.RLock()
		for k, _ := range s.managedNodes {
			updated, err := s.managedNodes[k].RingUpdate(msg.rb, msg.v)
			if err != nil {
				if updated {
					log.Printf("RingUpdate of %d succeeded but reported error: %v", k, err)
					continue
				}
				log.Printf("RingUpdate of %d failed: %v", k, err)
				continue
			}
			if !updated {
				log.Printf("RingUpdate of %d failed but reported no error", k)
				continue
			}
			log.Printf("RingUpdate of %d succeeded", k)
		}
		s.RUnlock()
	}
}

// TODO: if disconnect encounters an error we just log it and remove the node anyway
func (s *Server) removeManagedNode(nodeid uint64) {
	s.RLock()
	if node, ok := s.managedNodes[nodeid]; ok {
		node.Lock()
		s.RUnlock() // nothing else should be messing with s.managedNodes[nodeid] now
		err := node.Disconnect()
		if err != nil {
			log.Printf("Disconnect of node %d encountered err: %s", nodeid, err.Error())
		}
		s.Lock()
		delete(s.managedNodes, nodeid)
		s.Unlock()
		return
		//do something here
	} else {
		s.RUnlock()
		return
	}
}

// TODO: remove me, test func
func (s *Server) pingSweep() {
	responses := make(map[string]string, len(s.managedNodes))
	for k, _ := range s.managedNodes {
		_, msg, err := s.managedNodes[k].Ping()
		if err != nil {
			responses[s.managedNodes[k].address] = err.Error()
			continue
		}
		responses[s.managedNodes[k].address] = msg
	}
}
