package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	lowlevelfunctions "github.com/NikoMalik/low-level-functions"
	"github.com/NikoMalik/potoc/pkg/proto"
	"google.golang.org/grpc"
)

var (
	serverAddr string
	numThreads int
	filePath   string
	outputPath string
)

func main() {
	flag.StringVar(&serverAddr, "server", "localhost:50052", "Server address")
	flag.IntVar(&numThreads, "threads", runtime.GOMAXPROCS(runtime.NumCPU()), "Number of threads")
	flag.StringVar(&filePath, "file", "data.txt", "File to read data from")
	flag.StringVar(&outputPath, "output", "output.txt", "File to write received data from db")
	flag.Parse()

	var sendWg = &sync.WaitGroup{}
	var recvWg = &sync.WaitGroup{}
	str := make(chan string, numThreads)

	file, err := checkAndGenerateData(filePath, sendWg)
	if err != nil {
		log.Fatalf("Cancel open or generate data :%s", err.Error())
	}
	defer file.Close()

	fileOutput, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Failed to open output file: %s", err)
	}
	defer fileOutput.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	defer cancel()
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := proto.NewDataTranferClient(conn)

	for i := 0; i < numThreads; i++ {
		sendWg.Add(1)
		go func() {
			defer sendWg.Done()
			if s := sendData(ctx, client, file); s != "" {
				log.Printf("Sending socket ID: %s", s)
				str <- s
			}
		}()
	}

	go func() {
		sendWg.Wait()
		close(str)
		log.Println("All send goroutines completed.")
	}()

	for id := range str {
		log.Printf("Starting receive for socket ID: %s", id)
		recvWg.Add(1)
		go func(socketID string) {
			defer recvWg.Done()
			if err := receiveData(ctx, client, socketID, fileOutput); err != nil {
				log.Printf("Error receiving data for socket ID %s: %v", socketID, err)
			} else {
				log.Printf("Successfully received data for socket ID: %s", socketID)
			}
		}(id)
	}

	recvWg.Wait()
	fmt.Println("All threads completed.")
}

func sendData(ctx context.Context, client proto.DataTranferClient, file *os.File) string {

	stream, err := client.GetData(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	reader := bufio.NewScanner(file)
	var lastResp string

	for reader.Scan() {
		time.Sleep(100 * time.Millisecond)
		data := reader.Text()
		encodedData := base64.StdEncoding.EncodeToString([]byte(data))

		req := &proto.DataRequest{
			EncodedData: []byte(encodedData),
		}

		if err := stream.Send(req); err != nil {
			log.Printf("Failed to send data: %v", err)
			return ""
		}

		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Print("Stream ended")
				break
			}
			log.Printf("Failed to receive response: %v", err)
			return ""
		}

		log.Printf("Status: %s", resp.GetStatus())
		lastResp = lowlevelfunctions.String(resp.GetData())
		log.Printf("Return last resp success: %s", lastResp)
		return lastResp
	}

	log.Println("Closing stream after sending all data.")
	if err := stream.CloseSend(); err != nil {
		log.Printf("Failed to close stream: %v", err)
	}
	if err := reader.Err(); err != nil {
		log.Printf("Error while scanning: %v", err)
	}
	log.Println("Connection closed after all data sent.")
	return lastResp
}

func checkAndGenerateData(filePath string, wg *sync.WaitGroup) (*os.File, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		log.Fatalf("Failed to stat file: %v", err)
		return nil, err
	}

	if info.Size() == 0 {
		log.Println("File is empty. Generating random data...")

		writer := bufio.NewWriter(file)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				randomData := generateRandomData()
				writer.WriteString(randomData + "\n")
			}()
		}
		wg.Wait()
		writer.Flush()
		log.Println("Random data generation completed.")
	} else {
		log.Println("File is not empty. Proceeding with existing data.")
	}
	return file, nil
}

func generateRandomData() string {
	dataSize := rand.Intn(9000) + 1000
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(rand.Intn(26) + 65)
	}
	return lowlevelfunctions.String(data)
}

func receiveData(ctx context.Context, client proto.DataTranferClient, socketID string, file *os.File) error {
	stream, err := client.FetchData(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %s", err)
		return err
	}
	if socketID == "" {
		log.Fatalf("SocketId is empty, cannot send request")
		return fmt.Errorf("SocketId is empty")
	}

	log.Printf("Requesting data for SocketId: %s", socketID)

	writer := bufio.NewWriter(file)
	req := &proto.DataRequest{
		SocketId: socketID,
	}

	log.Printf("Sending request: SocketId=%s", req.GetSocketId())

	if err := stream.Send(req); err != nil {
		log.Fatalf("Failed to send request: %v", err)
		return err
	}
	time.Sleep(100 * time.Millisecond)

	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Println("Stream ended")
				break
			}
			log.Fatalf("Failed to receive response: %s", err)
			return err
		}

		log.Printf("Received data from server: %s, status: %s", lowlevelfunctions.String(resp.GetData()), resp.GetStatus())

		if resp.GetStatus() != "ok" {
			log.Println("Received empty message from server.")
			break
		}

		_, err = writer.WriteString(fmt.Sprintf("Received data: %s\n", lowlevelfunctions.String(resp.GetData())))
		if err != nil {
			log.Printf("Failed to write to file: %v", err)
			return err
		}
		log.Printf("Data written to file: %s", lowlevelfunctions.String(resp.GetData()))
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close stream: %s", err)
	}

	return nil
}
