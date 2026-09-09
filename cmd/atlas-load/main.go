package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const clientTimeout = 30 * time.Second

type result struct {
	requests int64
	errors   int64
	nanos    int64
}

func main() {
	addr := flag.String("addr", "127.0.0.1:6379", "Atlas address")
	clients := flag.Int("clients", 10, "number of concurrent clients")
	requests := flag.Int("requests", 1000, "requests per client")
	commandName := flag.String("command", "GET", "command to benchmark: GET or SET")
	flag.Parse()

	if *clients <= 0 || *requests <= 0 {
		fmt.Fprintln(os.Stderr, "clients and requests must be positive")
		os.Exit(2)
	}

	command := strings.ToUpper(*commandName)
	if command != "GET" && command != "SET" {
		fmt.Fprintln(os.Stderr, "command must be GET or SET")
		os.Exit(2)
	}

	if command == "GET" {
		if err := seed(*addr); err != nil {
			fmt.Fprintf(os.Stderr, "seed request failed: %v\n", err)
			os.Exit(1)
		}
	}

	start := time.Now()
	results := make(chan result, *clients)
	var wg sync.WaitGroup
	for i := 0; i < *clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- runClient(*addr, command, *requests)
		}()
	}
	wg.Wait()
	close(results)

	var total result
	for item := range results {
		total.requests += item.requests
		total.errors += item.errors
		total.nanos += item.nanos
	}

	duration := time.Since(start)
	throughput := float64(total.requests) / duration.Seconds()
	average := time.Duration(0)
	if total.requests > 0 {
		average = time.Duration(total.nanos / total.requests)
	}

	fmt.Printf("clients: %d\n", *clients)
	fmt.Printf("requests: %d\n", total.requests)
	fmt.Printf("errors: %d\n", total.errors)
	fmt.Printf("duration: %s\n", duration)
	fmt.Printf("throughput: %.2f requests/sec\n", throughput)
	fmt.Printf("average latency: %s\n", average)
}

func runClient(addr, commandName string, requests int) result {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return result{errors: int64(requests)}
	}
	defer conn.Close()

	// bound the session so a stalled server cannot hang the benchmark
	if err := conn.SetDeadline(time.Now().Add(clientTimeout)); err != nil {
		return result{errors: int64(requests)}
	}

	reader := bufio.NewReader(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		return result{errors: int64(requests)}
	}

	var output result
	for i := 0; i < requests; i++ {
		request := commandName + " load-key"
		if commandName == "SET" {
			request += " value"
		}
		request += "\n"

		start := time.Now()
		if _, err := conn.Write([]byte(request)); err != nil {
			output.errors++
			continue
		}
		if _, err := reader.ReadString('\n'); err != nil {
			output.errors++
			continue
		}

		output.requests++
		output.nanos += time.Since(start).Nanoseconds()
	}

	return output
}

func seed(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(clientTimeout)); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		return err
	}
	if _, err := conn.Write([]byte("SET load-key value\n")); err != nil {
		return err
	}
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if response != "OK\n" {
		return errors.New("unexpected SET response: " + response)
	}
	return nil
}
