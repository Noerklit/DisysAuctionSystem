package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	gRPC "github.com/Noerklit/DisysAuctionSystem/proto"
	"google.golang.org/grpc"
)

var nameOfBidder = flag.String("name", "default", "Senders name")
var tcpServer = flag.String("server", "5400", "Tcp server")

var _ports [5]string = [5]string{*tcpServer, "5401", "5402", "5403", "5404"}

var ctx context.Context
var servers []gRPC.AuctionSystemClient
var ServerConn map[gRPC.AuctionSystemClient]*grpc.ClientConn

func main() {
	flag.Parse()

	// Connect to log file
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("error opening file: %v\n", err)
		return
	}
	defer f.Close()
	log.SetOutput(f)

	ServerConn = make(map[gRPC.AuctionSystemClient]*grpc.ClientConn)
	joinServer()
	defer closeAllClients()

	//start the biding
	parseInput()
}

func closeAllClients() {
	for _, c := range ServerConn {
		c.Close()
	}
}

func joinServer() {
	//connect to server
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())
	for _, port := range _ports {
		timeContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn, err := grpc.DialContext(timeContext, fmt.Sprintf(":%s", port), opts...)
		if err != nil {
			log.Printf("Client failed to dial on port %s: %v\n", port, err)
			fmt.Printf("Client failed to dial on port %s: %v\n", port, err)
			continue
		}
		var s = gRPC.NewAuctionSystemClient(conn)
		servers = append(servers, s)
		ServerConn[s] = conn
	}
	ctx = context.Background()
}

func bid(amount string) {
	bidAmount, err := strconv.ParseInt(amount, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}

	amountOfBid := &gRPC.Amount{
		Amount:     bidAmount,
		BidderName: *nameOfBidder,
	}

	for _, s := range servers {
		if conReady(s) {
			ack, err := s.Bid(ctx, amountOfBid)
			if err != nil {
				log.Printf("Client %s: Bid failed, because of a lack of server response\n", *nameOfBidder)
				fmt.Printf("Client %s: Bid failed, because of a lack of server response\n", *nameOfBidder)
				log.Println(err)
			}
			switch ack.Message {
			case "Fail":
				fmt.Printf("Client %s: Bid failed, because amount was lower than current highest bid\n", *nameOfBidder)
				log.Printf("Client %s: Bid failed, because amount was lower than current highest bid\n", *nameOfBidder)
			case "Success":
				fmt.Printf("Client %s: Bid was successful\n", *nameOfBidder)
			default:
				fmt.Printf(ack.Message + "\n")
				log.Printf(ack.Message + "\n")

			}
		}

	}
}

func getResult() int64 {
	void := &gRPC.Void{} // Create an instance of a gRPC "Void" message
	for _, s := range servers {
		if conReady(s) { // Check if the connection to the server is ready
			outcome, _ := s.Result(ctx, void) // Call the "Result" method on the server
			if !outcome.Status {
				fmt.Printf("Client %s asked for the highest bid, which is %v\n", *nameOfBidder, outcome.HighestBid)
				log.Printf("Client %s asked for the highest bid, which is %v\n", *nameOfBidder, outcome.HighestBid)
			} else {
				fmt.Printf("The auction is over, the winner is: %v with a bid of: %v!\n", outcome.HighestBidder, outcome.HighestBid)
				log.Printf("The auction is over, the winner is: %v with a bid of: %v!\n", outcome.HighestBidder, outcome.HighestBid)
			}
			return outcome.HighestBid // Return the "HighestBid" from the result
		}
	}
	return -1 // Return -1 if no server is ready or an error occurs
}

func parseInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Type your bidding amount here or type \"status\" to get the current highest bid\n")
	fmt.Printf("--------------------\n")

	for {
		fmt.Printf("-> ")
		in, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			fmt.Printf("Please enter a valid number\n")
			return
		}
		in = strings.TrimSpace(in)
		if in == "status" {
			getResult()
		} else {
			bid(in)
		}
	}
}

func conReady(s gRPC.AuctionSystemClient) bool {
	return ServerConn[s].GetState().String() == "READY"
}
