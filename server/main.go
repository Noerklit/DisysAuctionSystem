package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	gRPC "github.com/Noerklit/DisysAuctionSystem/proto"
	"google.golang.org/grpc"
)

type Server struct {
	gRPC.UnimplementedAuctionSystemServer
	id               int64
	port             string
	ctx              context.Context
	highestBid       int64
	highestBidder    string
	AuctionIsOngoing bool
}

var server *Server
var counter = 0

var serverName = flag.String("name", "default", "Senders name")
var serverId = 0
var port = flag.String("port", "5400", "Server port")

var _ports [5]string = [5]string{*port, "5401", "5402", "5403", "5404"}

func main() {
	//Clears the log.txt file when a new server is started
	if counter == 0 {
		if err := os.Truncate("log.txt", 0); err != nil {
			log.Printf("Failed to truncate: %v\n", err)
			fmt.Printf("Failed to truncate: %v\n", err)
			counter++
		}
	}

	//connect to log file
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("error opening file: %v\n", err)
		return
	}
	defer f.Close()
	log.SetOutput(f)

	flag.Parse()
	go launchServer(_ports[:])
	go runAuction()
	for {
		time.Sleep(time.Second * 5)
	}

}

func runAuction() {
	durationOfAuction := time.Minute * 2
	time.Sleep(durationOfAuction)
	server.AuctionIsOngoing = false
	void := &gRPC.Void{}
	server.Result(server.ctx, void)
}

func newServer(serverPort string) *Server {
	server := &Server{
		id:               int64(serverId),
		port:             serverPort,
		ctx:              context.Background(),
		highestBid:       0,
		highestBidder:    "",
		AuctionIsOngoing: true,
	}
	serverId++
	return server
}

func launchServer(ports []string) {
	// Create listener tcp on given port or port 5400
	log.Printf("Server %s: is trying to create listener on port %s\n", *serverName, ports[0])
	fmt.Printf("Server %s: is trying to create listener on port %s\n", *serverName, ports[0])
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", ports[0]))
	if err != nil {
		log.Printf("Server %s: Failed to listen on port %s: %v\n", *serverName, ports[0], err)
		fmt.Printf("Server %s: Failed to listen on port %s: %v\n", *serverName, ports[0], err)
		if len(ports) > 1 {
			launchServer(ports[1:])
		} else {
			log.Printf("Server %s: Failed to find open port\n", *serverName)
			fmt.Printf("Server %s: Failed to find open port\n", *serverName)
			return
		}
	} else {
		log.Printf("Server %s: Listening on port %s\n", *serverName, ports[0])
		fmt.Printf("Server %s: Listening on port %s\n", *serverName, ports[0])
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	server = newServer(ports[0])
	gRPC.RegisterAuctionSystemServer(grpcServer, server)

	if err := grpcServer.Serve(list); err != nil {
		log.Printf("failed to server %v\n", err)
		fmt.Printf("failed to server %v\n", err)
		return
	}
}

func (s *Server) Bid(ctx context.Context, amount *gRPC.Amount) (*gRPC.Ack, error) {
	if server.AuctionIsOngoing {
		if amount.Amount > s.highestBid {
			s.highestBid = amount.Amount
			s.highestBidder = amount.BidderName
			fmt.Printf("Server %s: %v has the highest bid of %v\n", *serverName, s.highestBidder, s.highestBid)
			log.Printf("Server %s: %v has the highest bid of %v\n", *serverName, s.highestBidder, s.highestBid)
			return &gRPC.Ack{Message: "Success"}, nil
		}
		return &gRPC.Ack{Message: "Fail"}, nil
	} else {
		return &gRPC.Ack{Message: "The auction is not ongoing, why are you still bidding???"}, nil
	}
}

func (s *Server) Result(ctx context.Context, void *gRPC.Void) (*gRPC.Outcome, error) {
	if s.AuctionIsOngoing {
		return &gRPC.Outcome{Status: false, HighestBid: s.highestBid, HighestBidder: s.highestBidder}, nil
	} else {
		return &gRPC.Outcome{Status: true, HighestBid: s.highestBid, HighestBidder: s.highestBidder}, nil
	}
}
