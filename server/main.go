package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
	"math/rand"
	"flag"
	"os"

	gRPC "github.com/Noerklit/DisysAuctionSystem/proto"
	"google.golang.org/grpc"
)

type Server struct {
	gRPC.UnimplementedAuctionSystemServiceServer
	id               int32
	port			 string
	ctx              context.Context
	highestBid	   	 int32
	highestBidder 	 string
	AuctionIsOngoing bool
}
var server *Server

var serverName = flag.String("name", "default", "Senders name")
var serverId = 0
var port = flag.String("port", "5400", "Server port")

var _ports [5]string = [5]string{*port, "5401", "5402", "5403", "5404"}

func main() {
	//Clears the log.txt file when a new server is started
	if err := os.Truncate("log.txt", 0); err != nil {
		log.Printf("Failed to truncate: %v", err)
	}

	//connect to log file
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	flag.Parse()
	go launchServer(_ports[:])
	go runAuction()
	for {
		time.Sleep(time.Second*5)
	}

}

func runAuction() {
	durationOfAuction := time.Duration(rand.Intn(45) + 15) * time.Second
	time.Sleep(durationOfAuction)
	server.AuctionIsOngoing = false
	server.Result()
}

func newServer(serverPort string) *Server {
	server:= &Server{
		id:               int32(serverId),
		port:			  serverPort,
		ctx:              context.Background(),
		highestBid:		  0,
		highestBidder:	  "",
		AuctionIsOngoing: true,
	}
	serverId++
	return server
}

func launchServer(ports []string) {
	// Create listener tcp on given port or port 5400
	log.Printf("Server %s: is trying to create listener on port %s\n", *serverName, ports[0])
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", ports[0]))
	if err != nil {
		log.Printf("Server %s: Failed to listen on port %s: %v", *serverName, *port, err)
		if len(ports) > 1 {
			launchServer(ports[1:])
		} else {
			log.Fatalf("Server %s: Failed to find open port", *serverName)
		}
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	server = newServer(ports[0])
	gRPC.RegisterAuctionServiceServer(grpcServer, server)

	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to server %v", err)
	}
}

func(s *Server) Bid(ctx context.Context, amount *gRPC.Amount) (*gRPC.Ack, error){
	if server.AuctionIsOngoing {
		if amount.Amount > s.highestBid {
			s.highestBid = amount.Amount
			s.highestBidder = amount.bidderName
			fmt.Printf("Server %s: %v has the highest bid of %v\n",*serverName, s.highestBidder, s.highestBid)
			return &gRPC.Ack{message: "Success"}, nil
		}
		return &gRPC.Ack{message: "Fail"}, nil
	} else {
		return &gRPC.Ack{message: "Exception, the auction is not ongoing, why are you still bidding???"}, nil
	}
}


func(s *Server) Result(ctx context.Context, void *gRPC.Void) (*gRPC.Outcome, error){
	if s.AuctionIsOngoing {
		return &gRPC.Result{Outcome: "The auction is still ongoing, the highest bid is currently %d", s.highestBid}, nil
	} else {
		return &gRPC.Outcome{Outcome: "The auction is over, the winner is: %v with a bid of: %d!", s.highestBidder, s.highestBid}, nil
	}
}
