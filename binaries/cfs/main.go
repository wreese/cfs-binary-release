package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	pb "github.com/creiht/formic/proto"

	"bazil.org/fuse"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type server struct {
	fs *fs
	wg sync.WaitGroup
}

func newserver(fs *fs) *server {
	s := &server{
		fs: fs,
	}
	return s
}

func (s *server) serve() error {
	defer s.wg.Wait()

	for {
		req, err := s.fs.conn.ReadRequest()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.fs.handle(req)
		}()
	}
	return nil
}

func debuglog(msg interface{}) {
	fmt.Fprintf(os.Stderr, "%v\n", msg)
}

type rpc struct {
	conn *grpc.ClientConn
	api  pb.ApiClient
}

func newrpc(conn *grpc.ClientConn) *rpc {
	r := &rpc{
		conn: conn,
		api:  pb.NewApiClient(conn),
	}

	return r
}

type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

func main() {

	fusermountPath()
	flag.Usage = printUsage
	flag.Parse()
	clargs := getArgs(flag.Args())
	mountpoint := clargs["mountPoint"]
	serverAddr := clargs["host"]

	// Setup grpc
	var opts []grpc.DialOption
	creds := credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
	})
	opts = append(opts, grpc.WithTransportCredentials(creds))
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	// Uncomment the following to diable logs
	//log.SetOutput(new(NullWriter))

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("cfs"),
		fuse.Subtype("cfs"),
		fuse.LocalVolume(),
		fuse.VolumeName("CFS"),
		//fuse.AllowOther(),
		fuse.DefaultPermissions(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	rpc := newrpc(conn)
	fs := newfs(c, rpc)
	srv := newserver(fs)

	if err := srv.serve(); err != nil {
		log.Fatal(err)
	}

	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}

// getArgs is passed a command line and breaks it up into commands
// the valid format is <device> <mount point> -o [Options]
func getArgs(args []string) map[string]string {
	// Setup declarations
	var optList []string
	requiredOptions := []string{"host"}
	clargs := make(map[string]string)

	// Not the correct number of arguments or -help
	if len(args) != 4 {
		printUsage()
		os.Exit(0)
	}

	// Verify mountPoint exists
	if _, err := os.Stat(args[1]); os.IsNotExist(err) {
		printUsage()
		log.Fatalf("Mount point %s does not exist\n\n", args[1])
	}

	// process options -o
	if args[2] == "-o" || args[2] == "--o" {
		optList = strings.Split(args[3], ",")
		for _, item := range optList {
			if strings.Contains(item, "=") {
				value := strings.Split(item, "=")
				if value[0] == "" || value[1] == "" {
					printUsage()
					log.Fatalf("Invalid option %s, %s no value\n\n", value[0], value[1])
				} else {
					clargs[value[0]] = value[1]
				}
			} else {
				clargs[item] = ""
			}
		}
	} else {
		printUsage()
		log.Fatalf("Invalid option %v\n\n", args[2])
	}

	// Verify required options exist
	for _, v := range requiredOptions {
		_, ok := clargs[v]
		if !ok {
			printUsage()
			log.Fatalf("%s is a required option", v)
		}
	}

	// load in device and mountPoint
	clargs["cfsDevice"] = args[0]
	clargs["mountPoint"] = args[1]
	return clargs
}

func fusermountPath() {
	// Grab the current path
	currentPath := os.Getenv("PATH")
	if len(currentPath) == 0 {
		// using mount seem to not have a path
		// fusermount is in /bin
		os.Setenv("PATH", "/bin")
	}
}

// printUsage will display usage
func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("\tcfs [file system] [mount point] -o [OPTIONS] -help")
	fmt.Println("\texample: mount -t cfs fsaas /mnt/cfsdrive -o host=localhost:8445")
	fmt.Println("\tMount Options: (separated by commas with no spaces)")
	fmt.Println("\t\tRequired:")
	fmt.Println("\t\t\thost\taddress of the formic server")
	fmt.Println("")
	fmt.Println("\thelp\tdisplay usage")
}
