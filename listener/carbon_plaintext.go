// Package listener implements the listener pool and handlers for metrics.
package listener

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	// "github.com/jeffpierce/cassabon/logging"
)

// Define CarbonMetric struct
type CarbonMetric struct {
	Path      string  // Metric path
	Value     float64 // Metric Value
	Timestamp float64 // Epoch timestamp
}

func CarbonTCP(addr string, port int) {
	// Test if we should use TCP or UDP.
	carbonTCPSocket, err := net.Listen("tcp", addr+":"+strconv.Itoa(port))
	if err != nil {
		// If we can't grab a port, we can't do our job.  Log, whine, and crash.
		// TODO: Convert to our own logger, add a stat.
		panic(err)
	}

	defer carbonTCPSocket.Close()

	// TODO:  Convert to our own logger.
	fmt.Printf("Carbon TCP plaintext listener now listening on %s:%d\n", addr, port)

	// Start listener and pass incoming connections to handler.
ListenerLoop:
	for {
		conn, err := carbonTCPSocket.Accept()
		if err != nil {
			// TODO: Log inability to handle connection.
			continue ListenerLoop
		}

		// Pass to handler to place metrics in queue.
		go MetricHandler(conn)
	}
}

// UDP totally blocks hard.  Need to figure this out. -- Jeff 2015/08/14

/* func CarbonUDP(addr string, port int) {
	udpaddr := net.UDPAddr{Port: port, IP: net.ParseIP(addr)}
	carbonUDPSocket, err := net.ListenUDP("udp", &udpaddr)
	if err != nil {
		// TODO:  Move to our own logger.
		panic(err)
	}

	defer carbonUDPSocket.Close()

	fmt.Printf("Carbon UDP plaintext listener now listening on %s:%d\n", addr, port)

	for {
		go MetricHandler(carbonUDPSocket)
	}
} */

func MetricHandler(conn net.Conn) {
	// Carbon metrics are terminated by newlines.  Listed for it.
	metric, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		// TODO:  Handle with actual logger/stats.
		fmt.Println("Received this error:", err.Error())
		metric = ""
	}

	// Close connection.
	conn.Close()

	// Examine metric to ensure that it's a valid carbon metric
	for len(metric) != 0 {
		splitMetric := strings.Fields(metric)
		if len(splitMetric) != 3 {
			// TODO:  Handle with actual logger/stats.
			fmt.Println("Bad metric:", metric)
			metric = ""
		}

		statPath := splitMetric[0]
		val, err := strconv.ParseFloat(splitMetric[1], 64)
		if err != nil {
			// TODO:  Handle with actual logger/stats.
			fmt.Printf("Cannot convert value %s to float.\n", splitMetric[1])
			break
		}
		ts, err := strconv.ParseFloat(splitMetric[2], 64)
		if err != nil {
			// TODO:  Handle with actual logger/stats.
			fmt.Printf("Cannot convert timestamp %s to float.\n", splitMetric[2])
			break
		}

		parsedMetric := CarbonMetric{statPath, val, ts}

		// Metric parsed, place in queue, handoff to receiving worker.
		// TODO:  Implement receiving worker
		fmt.Printf("Would queue parsed metric: %+v\n", parsedMetric)
		break
	}
}