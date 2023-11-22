package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"flag"
	"bufio"
	"strings"

	gRPC "github.com/Noerklit/DisysAuctionSystem/proto"
	"google.golang.org/grpc"
)

var nameOfBidder = flag.String("name", "default", "Senders name")
var tcpServer = flag.String("server", "5400", "Tcp server")

var _ports [5]string = [5]string{*tcpServer, "5401", "5402", "5403", "5404"}

var ctx context.Context
var servers []gRPC.AuctionSystemServiceClient
var ServerConn map[gRPC.AuctionSystemServiceClient]*grpc.ClientConn

func main() {
	flag.Parse()

	// Connect to log file
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	ServerConn = make(map[gRPC.AuctionServiceClient]*grpc.ClientConn)
	joinServer()
	defer closeAllClients()

	//start the biding
	parseInput()
}

func closeAllClients()  {
	for _, c := range ServerConn {
		c.Close()
	}
}

func joinServer() {
	//connect to server
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())
	timeContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	for _, port := range _ports {
		log.Printf("client %s: Attempts to dial on port %s\n", *nameOfBidder, port)
		conn, err := grpc.DialContext(timeContext, fmt.Sprintf(":%s", port), opts...)
		if err != nil {
			log.Printf("Fail to Dial : %v", err)
			continue
		}
		var s = gRPC.NewAuctionSystemServiceClient(conn)
		servers = append(servers, s)
		ServerConn[s] = conn
	}
	ctx = context.Background()
}

func bid(amount string) {
	for {
		bidAmount, err := strconv.Atoi(amount)
		if err != nil {
			log.Fatal(err)
		}

		amount := &gRPC.Amount{
			Amount: bidAmount,
			nameOfBidder: *nameOfBidder,
		}

		for _, s := range servers {
			if conReady(s) {
				ack, err := s.Bid(ctx, amount)
				if err != nil {
					log.Printf("Client: %s' bid failed, because of a lack of server response", *nameOfBidder, err)
				}
				switch ack.Message {
				case "Fail":
					fmt.Printf("Client: %s' bid failed, because amount was lower than current highest bid", *nameOfBidder)
					log.Printf("Client: %s' bid failed, because amount was lower than current highest bid", *nameOfBidder)
				case "Success":
					fmt.Printf("Client: %s' bid was successful", *nameOfBidder)
					log.Printf("Client: %s' bid was successful", *nameOfBidder)
				default:
					fmt.Printf(ack.Message)
					log.Printf(ack.Message)

				}
			}
		
		}
		parseInput()
	}
}

func getResult() int32 {
    void := &gRPC.Void{}  // Create an instance of a gRPC "Void" message
    for _, s := range servers {
        if conReady(s) {  // Check if the connection to the server is ready
            outcome, _ := s.Result(ctx, void)  // Call the "Result" method on the server
            return outcome.HighestBid  // Return the "HighestBid" from the result
        }
    }
    return -1  // Return -1 if no server is ready or an error occurs
}


func parseInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Type your bidding amount here or type \"status\" to get the current highest bid")
	fmt.Println("--------------------")

	for {
		fmt.Print("-> ")
		in, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		in = strings.TrimSpace(in) 
		if in == "status" {
			fmt.Printf("The current highest bid is %d\n", getResult())
		} else {
			bid(in)
		}
	}
}

func conReady(s gRPC.AuctionServiceClient) bool {
	return ServerConn[s].GetState().String() == "READY"
}