package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	pbStorage "orbital-net/pkg/proto/storage"

	"google.golang.org/grpc"
)

const PORT = "50054"
const LOG_FILE = "/data/messages.log"

type Server struct {
	pbStorage.UnimplementedStorageServiceServer
	mu sync.Mutex
}

func (s *Server) SaveMessage(ctx context.Context, req *pbStorage.SaveRequest) (*pbStorage.SaveResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := fmt.Sprintf("%d|%s|%s|%s\n", req.Timestamp, req.SenderId, req.ReceiverId, req.Payload)

	f, err := os.OpenFile(LOG_FILE, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("❌ Error escribiendo archivo: %v", err)
		return &pbStorage.SaveResponse{Success: false, Message: "Error de disco"}, nil
	}
	defer f.Close()

	if _, err := f.WriteString(entry); err != nil {
		return &pbStorage.SaveResponse{Success: false}, nil
	}

	log.Printf("💾 Mensaje guardado: %s", req.Payload)
	return &pbStorage.SaveResponse{Success: true}, nil
}

func (s *Server) GetLogs(ctx context.Context, _ *pbStorage.Empty) (*pbStorage.LogList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(LOG_FILE)
	if err != nil {
		return &pbStorage.LogList{}, nil
	}
	defer file.Close()

	var ringBuffer []*pbStorage.LogEntry

	maxLogs := 25

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			entry := &pbStorage.LogEntry{
				SenderId:   parts[1],
				ReceiverId: parts[2],
				Payload:    parts[3],
			}

			ringBuffer = append(ringBuffer, entry)
			if len(ringBuffer) > maxLogs {
				ringBuffer = ringBuffer[1:]
			}
		}
	}

	return &pbStorage.LogList{Entries: ringBuffer}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":"+PORT)
	if err != nil {
		log.Fatal(err)
	}

	os.MkdirAll("/data", 0755)

	s := grpc.NewServer()
	pbStorage.RegisterStorageServiceServer(s, &Server{})

	log.Printf("💾 Storage Service corriendo en puerto %s", PORT)
	s.Serve(lis)
}
