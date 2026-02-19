package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/sokoide/workshop/infra/assets/http_persistent_conn/pb"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// HTTP/1.1 & WebSocket Handler
func handler(w http.ResponseWriter, r *http.Request) {
	if websocket.IsWebSocketUpgrade(r) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		defer conn.Close()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", message)
			err = conn.WriteMessage(mt, message)
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
		return
	}

	// Normal HTTP/1.1
	log.Printf("HTTP/1.1 request from %s", r.RemoteAddr)
	fmt.Fprintf(w, "Hello from HTTP/1.1! Protocol: %s\n", r.Proto)
}

// SSE Handler
func sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	log.Printf("SSE client connected from %s", r.RemoteAddr)
	for i := 0; i < 10; i++ {
		fmt.Fprintf(w, "data: Message %d at %s\n\n", i, time.Now().Format(time.RFC3339))
		flusher.Flush()
		time.Sleep(2 * time.Second)
	}
}

// gRPC Server Implementation
type greeterServer struct {
	pb.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func (s *greeterServer) SayHelloStream(in *pb.HelloRequest, stream pb.Greeter_SayHelloStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&pb.HelloReply{Message: fmt.Sprintf("Hello %s (%d)", in.Name, i)}); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func (s *greeterServer) Chat(stream pb.Greeter_ChatServer) error {
	for {
		in, err := stream.Recv()
		if err != nil {
			return err
		}
		log.Printf("gRPC Chat recv: %s", in.Name)
		if err := stream.Send(&pb.HelloReply{Message: "Echo: " + in.Name}); err != nil {
			return err
		}
	}
}

func main() {
	// 1. HTTP/1.1, WebSocket & SSE on :8080
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", handler)
		mux.HandleFunc("/sse", sseHandler)
		log.Println("Starting HTTP/1.1, WebSocket & SSE server on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal(err)
		}
	}()

	// 2. HTTP/2 on :8443 (TLS required)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP/2 request from %s, Proto: %s", r.RemoteAddr, r.Proto)
			fmt.Fprintf(w, "Hello from %s!\n", r.Proto)
		})

		server := &http.Server{
			Addr:    ":8443",
			Handler: mux,
		}
		log.Println("Starting HTTP/2 server on :8443 (TLS required)")
		if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			log.Printf("HTTP/2 server failed: %v", err)
		}
	}()

	// 3. HTTP/3 on :8444 (QUIC)
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("HTTP/3 request from %s, Proto: %s", r.RemoteAddr, r.Proto)
			fmt.Fprintf(w, "Hello from %s (QUIC)!\n", r.Proto)
		})

		server := &http3.Server{
			Addr:    ":8444",
			Handler: mux,
		}
		log.Println("Starting HTTP/3 server on :8444 (QUIC)")
		if err := server.ListenAndServeTLS("server.crt", "server.key"); err != nil {
			log.Printf("HTTP/3 server failed: %v", err)
		}
	}()

	// 4. gRPC on :50051
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterGreeterServer(s, &greeterServer{})
		reflection.Register(s)
		log.Println("Starting gRPC server on :50051")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	select {} // Keep running
}
