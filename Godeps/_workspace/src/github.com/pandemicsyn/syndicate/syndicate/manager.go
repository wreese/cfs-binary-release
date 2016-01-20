package syndicate

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gholt/brimtext"
	"github.com/gholt/ring"
	pb "github.com/pandemicsyn/syndicate/api/proto"
	"golang.org/x/net/context"
)

const (
	_SYN_REGISTER_TIMEOUT = 4
	_SYN_DIAL_TIMEOUT     = 2
	DefaultPort           = 8443
	DefaultCmdCtrlPort    = 4443
	DefaultMsgRingPort    = 8001
	DefaultRingDir        = "/etc/oort/ring"
	DefaultCertFile       = "/etc/oort/server.crt"
	DefaultCertKey        = "/etc/oort/server.key"
)

var (
	DefaultNetFilter  = []string{"10.0.0.0/8", "192.168.0.0/16"} //need to pull from conf
	DefaultTierFilter = []string{".*"}
)

type Config struct {
	Master           bool
	Slaves           []string
	NetFilter        []string
	TierFilter       []string
	Port             int
	MsgRingPort      int
	CmdCtrlPort      int
	RingDir          string
	CertFile         string
	KeyFile          string
	WeightAssignment string
}

func parseSlaveAddrs(slaveAddrs []string) []*RingSlave {
	slaves := make([]*RingSlave, len(slaveAddrs))
	for i, v := range slaveAddrs {
		slaves[i] = &RingSlave{
			status: false,
			addr:   v,
		}
	}
	return slaves
}

type Server struct {
	sync.RWMutex
	servicename  string
	cfg          *Config
	r            ring.Ring
	b            *ring.Builder
	slaves       []*RingSlave
	localAddress string
	rb           *[]byte // even a 1000 node ring is reasonably small (17k) so just keep the current ring in mem
	bb           *[]byte
	netlimits    []*net.IPNet
	tierlimits   []string
	managedNodes map[uint64]*ManagedNode
	changeChan   chan *changeMsg
}

func NewServer(cfg *Config, servicename string) (*Server, error) {
	var err error
	s := new(Server)
	s.cfg = cfg
	s.servicename = servicename
	s.parseConfig()

	bfile, rfile, err := getRingPaths(cfg, s.servicename)
	if err != nil {
		panic(err)
	}
	_, s.b, err = ring.RingOrBuilder(bfile)
	FatalIf(err, fmt.Sprintf("Builder file (%s) load failed:", bfile))
	s.r, _, err = ring.RingOrBuilder(rfile)
	FatalIf(err, fmt.Sprintf("Ring file (%s) load failed:", rfile))
	log.Println("Ring version is:", s.r.Version())
	//TODO: verify ring version in bytes matches what we expect
	s.rb, s.bb, err = s.loadRingBuilderBytes(s.r.Version())
	FatalIf(err, "Attempting to load ring/builder bytes")

	for _, v := range cfg.NetFilter {
		_, n, err := net.ParseCIDR(v)
		if err != nil {
			FatalIf(err, "Invalid network range provided")
		}
		s.netlimits = append(s.netlimits, n)
	}
	s.tierlimits = cfg.TierFilter
	s.managedNodes = bootstrapManagedNodes(s.r, s.cfg.CmdCtrlPort)
	s.changeChan = make(chan *changeMsg, 1)
	go s.RingChangeManager()
	s.slaves = parseSlaveAddrs(cfg.Slaves)
	if len(s.slaves) == 0 {
		log.Println("!! Running without slaves, have no one to register !!")
		return s, nil
	}

	failcount := 0
	for _, slave := range s.slaves {
		if err = s.RegisterSlave(slave); err != nil {
			log.Println("Got error:", err)
			failcount++
		}
	}
	if failcount > (len(s.slaves) / 2) {
		log.Fatalln("More than half of the ring slaves failed to respond. Exiting.")
	}
	return s, nil
}

func (s *Server) parseConfig() {
	if s.cfg.NetFilter == nil {
		s.cfg.NetFilter = DefaultNetFilter
		log.Println("Config didn't specify netfilter, using default:", DefaultNetFilter)
	}
	if s.cfg.TierFilter == nil {
		s.cfg.TierFilter = DefaultTierFilter
		log.Println("Config didn't specify tierfilter, using default:", DefaultTierFilter)
	}

	if s.cfg.Port == 0 {
		log.Println("Config didn't specify port, using default:", DefaultPort)
		s.cfg.Port = DefaultPort
	}
	if s.cfg.MsgRingPort == 0 {
		log.Println("Config didn't specify ring port, using default:", DefaultPort)
		s.cfg.MsgRingPort = DefaultMsgRingPort
	}
	if s.cfg.CmdCtrlPort == 0 {
		log.Println("Config didn't specify cmdctrl port, using default:", DefaultCmdCtrlPort)
		s.cfg.Port = DefaultCmdCtrlPort
	}
	if s.cfg.RingDir == "" {
		s.cfg.RingDir = filepath.Join(DefaultRingDir, s.servicename)
		log.Println("Config didn't specify ringdir, using default:", s.cfg.RingDir)

	}
	if s.cfg.CertFile == "" {
		log.Println("Config didn't specify certfile, using default:", DefaultCertFile)
		s.cfg.CertFile = DefaultCertFile
	}
	if s.cfg.KeyFile == "" {
		log.Println("Config didn't specify keyfile, using default:", DefaultCertKey)
		s.cfg.KeyFile = DefaultCertKey
	}
}

func (s *Server) loadRingBuilderBytes(version int64) (ring, builder *[]byte, err error) {
	b, err := ioutil.ReadFile(fmt.Sprintf("%s/%d-%s.builder", s.cfg.RingDir, version, s.servicename))
	if err != nil {
		return ring, builder, err
	}
	r, err := ioutil.ReadFile(fmt.Sprintf("%s/%d-%s.ring", s.cfg.RingDir, version, s.servicename))
	if err != nil {
		return ring, builder, err
	}
	return &r, &b, nil
}

type ringChange struct {
	b *ring.Builder
	r ring.Ring
	v int64
}

func (s *Server) applyRingChange(c *ringChange) error {
	if err := ring.PersistRingOrBuilder(nil, c.b, fmt.Sprintf("%s/%d-%s.builder", s.cfg.RingDir, c.v, s.servicename)); err != nil {
		return err
	}
	if err := ring.PersistRingOrBuilder(c.r, nil, fmt.Sprintf("%s/%d-%s.ring", s.cfg.RingDir, c.v, s.servicename)); err != nil {
		return err
	}
	newRB, newBB, err := s.loadRingBuilderBytes(c.v)
	if err != nil {
		return fmt.Errorf("Failed to load new ring/builder bytes: %s", err)
	}
	err = s.replicateRing(c.r, newRB, newBB)
	if err != nil {
		return fmt.Errorf("Ring replicate failed: %s", err)
	}
	if err := ring.PersistRingOrBuilder(nil, c.b, fmt.Sprintf("%s/%s.builder", s.cfg.RingDir, s.servicename)); err != nil {
		log.Println("Unable to persist builder!")
		return err
	}
	if err := ring.PersistRingOrBuilder(c.r, nil, fmt.Sprintf("%s/%s.ring", s.cfg.RingDir, s.servicename)); err != nil {
		log.Println("Unable to persist ring!")
		return err
	}
	s.rb = newRB
	s.bb = newBB
	s.b = c.b
	s.r = c.r
	go s.NotifyNodes()
	return nil
}

// TODO: Need field/value error checks
func (s *Server) AddNode(c context.Context, e *pb.Node) (*pb.RingStatus, error) {
	s.Lock()
	defer s.Unlock()
	log.Println("Got AddNode request")
	_, b, err := ring.RingOrBuilder(fmt.Sprintf("%s/%s.builder", s.cfg.RingDir, s.servicename))
	if err != nil {
		log.Println("Unable to load builder for change:", err)
		return &pb.RingStatus{}, err
	}
	n, err := b.AddNode(e.Active, e.Capacity, e.Tiers, e.Addresses, e.Meta, e.Conf)
	if err != nil {
		return &pb.RingStatus{}, err
	}
	report := [][]string{
		[]string{"ID:", fmt.Sprintf("%016x", n.ID())},
		[]string{"RAW ID", fmt.Sprintf("%d", n.ID())},
		[]string{"Active:", fmt.Sprintf("%v", n.Active())},
		[]string{"Capacity:", fmt.Sprintf("%d", n.Capacity())},
		[]string{"Tiers:", strings.Join(n.Tiers(), "\n")},
		[]string{"Addresses:", strings.Join(n.Addresses(), "\n")},
		[]string{"Meta:", n.Meta()},
		[]string{"Conf:", fmt.Sprintf("%s", n.Conf())},
	}
	log.Print(brimtext.Align(report, nil))
	newRing := b.Ring()
	log.Println("Attempting to apply ring version:", newRing.Version())
	err = s.applyRingChange(&ringChange{b: b, r: newRing, v: newRing.Version()})
	if err != nil {
		log.Println("Failed to apply ring change:", err)
	}
	log.Println("Ring version is now:", s.r.Version())

	return &pb.RingStatus{Status: true, Version: s.r.Version()}, err
}

func (s *Server) RemoveNode(c context.Context, n *pb.Node) (*pb.RingStatus, error) {
	s.Lock()
	defer s.Unlock()
	log.Println("Got RemoveNode request for:", n.Id)
	_, b, err := ring.RingOrBuilder(fmt.Sprintf("%s/%s.builder", s.cfg.RingDir, s.servicename))
	if err != nil {
		log.Println("Unable to load builder for change:", err)
		return &pb.RingStatus{}, err
	}
	node := b.Node(n.Id)
	if node == nil {
		return &pb.RingStatus{Status: true, Version: s.r.Version()}, fmt.Errorf("Node ID not found")
	}
	b.RemoveNode(n.Id)
	newRing := b.Ring()
	go s.removeManagedNode(n.Id)
	log.Println("Attempting to apply ring version:", newRing.Version())
	err = s.applyRingChange(&ringChange{b: b, r: newRing, v: newRing.Version()})
	if err != nil {
		log.Println(" Failed to apply ring change:", err)
	}
	log.Println("Ring version is now:", s.r.Version())
	return &pb.RingStatus{Status: true, Version: s.r.Version()}, err
}

func (s *Server) ModNode(c context.Context, n *pb.ModifyMsg) (*pb.RingStatus, error) {
	return &pb.RingStatus{}, nil
}

func (s *Server) SetConf(c context.Context, conf *pb.Conf) (*pb.RingStatus, error) {
	s.Lock()
	defer s.Unlock()
	log.Println("Got SetConf request")
	_, b, err := ring.RingOrBuilder(fmt.Sprintf("%s/%s.builder", s.cfg.RingDir, s.servicename))
	if err != nil {
		log.Println("Unable to load builder for change:", err)
		return &pb.RingStatus{}, err
	}
	b.SetConf(conf.Conf)
	newRing := b.Ring()
	log.Println("Attempting to apply ring version:", newRing.Version())
	err = s.applyRingChange(&ringChange{b: b, r: newRing, v: newRing.Version()})
	if err != nil {
		log.Println("Failed to apply ring change:", err)
	}
	log.Println("Ring version is now:", s.r.Version())
	return &pb.RingStatus{Status: true, Version: s.r.Version()}, err
}

func (s *Server) SetActive(c context.Context, n *pb.Node) (*pb.RingStatus, error) {
	s.Lock()
	defer s.Unlock()
	return &pb.RingStatus{Status: true, Version: s.r.Version()}, nil
}

func (s *Server) GetVersion(c context.Context, n *pb.EmptyMsg) (*pb.RingStatus, error) {
	s.RLock()
	defer s.RUnlock()
	return &pb.RingStatus{Status: true, Version: s.r.Version()}, nil
}

func (s *Server) GetGlobalConfig(c context.Context, n *pb.EmptyMsg) (*pb.RingConf, error) {
	s.RLock()
	defer s.RUnlock()
	config := &pb.RingConf{
		Status: &pb.RingStatus{Status: true, Version: s.r.Version()},
		Conf:   &pb.Conf{Conf: s.r.Conf(), RestartRequired: false},
	}
	return config, nil
}

func (s *Server) SearchNodes(c context.Context, n *pb.Node) (*pb.SearchResult, error) {
	s.RLock()
	defer s.RUnlock()

	var filter []string

	if n.Id != 0 {
		filter = append(filter, fmt.Sprintf("id=%d", n.Id))
	}
	if n.Meta != "" {
		filter = append(filter, fmt.Sprintf("meta~=%s.*", n.Meta))
	}
	if len(n.Tiers) >= 1 {
		for _, v := range n.Tiers {
			filter = append(filter, fmt.Sprintf("tier~=%s.*", v))
		}
	}
	if len(n.Addresses) >= 1 {
		for _, v := range n.Addresses {
			filter = append(filter, fmt.Sprintf("address~=%s.*", v))
		}
	}
	nodes, err := s.r.Nodes().Filter(filter)
	res := make([]*pb.Node, len(nodes))
	if err != nil {
		return &pb.SearchResult{Nodes: res}, err
	}
	for i, n := range nodes {
		if n == nil {
			continue
		}
		res[i] = &pb.Node{
			Id:        n.ID(),
			Active:    n.Active(),
			Capacity:  n.Capacity(),
			Tiers:     n.Tiers(),
			Addresses: n.Addresses(),
			Meta:      n.Meta(),
			Conf:      n.Conf(),
		}
	}
	return &pb.SearchResult{Nodes: res}, nil
}

func (s *Server) GetNodeConfig(c context.Context, n *pb.Node) (*pb.RingConf, error) {
	s.RLock()
	defer s.RUnlock()
	node := s.r.Node(n.Id)
	if node == nil {
		return &pb.RingConf{}, fmt.Errorf("Node %d not found", n.Id)
	}

	config := &pb.RingConf{
		Status: &pb.RingStatus{Status: true, Version: s.r.Version()},
		Conf:   &pb.Conf{Conf: s.r.Conf(), RestartRequired: false},
	}
	return config, nil
}

func (s *Server) GetRing(c context.Context, e *pb.EmptyMsg) (*pb.Ring, error) {
	s.RLock()
	defer s.RUnlock()
	return &pb.Ring{Ring: *s.rb}, nil
}

// validNodeIP verifies that the provided ip is not a loopback or multicast address
// and checks whether the ip is in the configured network limits range.
func (s *Server) validNodeIP(i net.IP) bool {
	switch {
	case i.IsLoopback():
		return false
	case i.IsMulticast():
		return false
	}
	inRange := false
	for _, n := range s.netlimits {
		if n.Contains(i) {
			inRange = true
		}
	}
	return inRange
}

// tier0 must never already exist as a tier0 entry in the ring
func (s *Server) validTiers(t []string) bool {
	if len(t) == 0 {
		return false
	}
	r, err := s.r.Nodes().Filter([]string{fmt.Sprintf("tier0=%s", t[0])})
	if len(r) != 0 || err != nil {
		return false
	}
	/*
		//we're not using multiple tiers anymore
		for i := 1; i <= len(t); i++ {
			for _, v := range s.tierlimits {
				matched, err := regexp.MatchString(v, t[i])
				if err != nil {
					return false
				}
				if matched {
					return true
				}
			}
		}
	*/
	return true
}

// nodeInRing just checks to see if the hostname or addresses appear
// in any existing entries meta or address fields.
func (s *Server) nodeInRing(hostname string, addrs []string) bool {
	a := strings.Join(addrs, "|")
	r, _ := s.r.Nodes().Filter([]string{fmt.Sprintf("meta~=%s.*", hostname), fmt.Sprintf("address~=%s", a)})
	if len(r) != 0 {
		return true
	}
	return false
}

func (s *Server) RegisterNode(c context.Context, r *pb.RegisterRequest) (*pb.NodeConfig, error) {
	s.Lock()
	defer s.Unlock()
	log.Printf("Got Register request: %#v", r)
	_, b, err := ring.RingOrBuilder(fmt.Sprintf("%s/%s.builder", s.cfg.RingDir, s.servicename))
	if err != nil {
		log.Println("Unable to load builder for change:", err)
		return &pb.NodeConfig{}, err
	}

	var addrs []string
	for _, v := range r.Addrs {
		i, _, err := net.ParseCIDR(v)
		if err != nil {
			log.Println("Encountered unknown network addr", v, err)
			continue
		}
		if s.validNodeIP(i) {
			addrs = append(addrs, fmt.Sprintf("%s:%d", i.String(), s.cfg.MsgRingPort))
		}
	}
	switch {
	case len(addrs) == 0:
		log.Println("Host provided no valid addresses during registration.")
		return &pb.NodeConfig{}, fmt.Errorf("No valid addresses provided")
	case s.nodeInRing(r.Hostname, addrs):
		a := strings.Join(addrs, "|")
		r, _ := s.r.Nodes().Filter([]string{fmt.Sprintf("meta~=%s.*", r.Hostname), fmt.Sprintf("address~=%s", a)})
		if len(r) > 1 {
			log.Println("Found more than one match when attempting to find node ID:", r)
			return &pb.NodeConfig{}, fmt.Errorf("Node already in ring/unable to obtain ID")
		}
		log.Println("Node already in ring, sending localid:", r[0].ID())
		return &pb.NodeConfig{Localid: r[0].ID(), Ring: *s.rb}, nil
	case !s.validTiers(r.Tiers):
		return &pb.NodeConfig{}, fmt.Errorf("Invalid tiers provided")
	}

	var weight uint32
	nodeEnabled := false

	if s.cfg.WeightAssignment == "fixed" {
		weight = 1000
		nodeEnabled = true
	}
	if s.cfg.WeightAssignment == "self" {
		log.Println("SELF weight assignment strategy no longer implemented!")
		weight = 1000
		nodeEnabled = true
	}
	if s.cfg.WeightAssignment == "manual" {
		log.Println("MANUAL weight assignment strategy no longer implemented!")
		weight = 1000
		nodeEnabled = true
	}
	n, err := b.AddNode(nodeEnabled, weight, r.Tiers, addrs, fmt.Sprintf("%s|%s", r.Hostname, r.Hardwareid), []byte(""))
	if err != nil {
		return &pb.NodeConfig{}, err
	}

	report := [][]string{
		[]string{"ID:", fmt.Sprintf("%016x", n.ID())},
		[]string{"RAW ID", fmt.Sprintf("%d", n.ID())},
		[]string{"Active:", fmt.Sprintf("%v", n.Active())},
		[]string{"Capacity:", fmt.Sprintf("%d", n.Capacity())},
		[]string{"Tiers:", strings.Join(n.Tiers(), "\n")},
		[]string{"Addresses:", strings.Join(n.Addresses(), "\n")},
		[]string{"Meta:", n.Meta()},
		[]string{"Conf:", fmt.Sprintf("%s", n.Conf())},
	}
	log.Print(brimtext.Align(report, nil))
	newRing := b.Ring()
	log.Println("Attempting to apply ring version:", newRing.Version())
	err = s.applyRingChange(&ringChange{b: b, r: newRing, v: newRing.Version()})
	if err != nil {
		log.Println("Failed to apply ring change:", err)
		log.Println("Ring version is now:", s.r.Version())
		return &pb.NodeConfig{}, fmt.Errorf("Unable to apply ring change during registration")
	}
	maddr, _ := ParseManagedNodeAddress(n.Address(0), s.cfg.CmdCtrlPort)
	s.managedNodes[n.ID()], err = NewManagedNode(maddr)
	//just log the error, we'll keep retrying to connect
	if err != nil {
		log.Printf("Error setting up new managed node %s: %s", n.Address(0), err.Error())
	}
	log.Printf("Added node %d ring version is now %d", n.ID(), s.r.Version())
	return &pb.NodeConfig{Localid: n.ID(), Ring: *s.rb}, nil
}
