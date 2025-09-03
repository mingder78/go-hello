package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    iplog "github.com/ipfs/go-log/v2"
    "github.com/libp2p/go-libp2p"
    "github.com/libp2p/go-libp2p/core/network"
    "github.com/libp2p/go-libp2p/core/peer"
    dht "github.com/libp2p/go-libp2p-kad-dht"
    ma "github.com/multiformats/go-multiaddr"
)

func main() {
    // Set up libp2p logging to debug level
    iplog.SetLogLevel("dht", "DEBUG")    // Debug logging for DHT
    iplog.SetLogLevel("libp2p", "DEBUG") // Debug logging for libp2p core
    iplog.SetLogLevel("net", "DEBUG")    // Debug logging for network layer

    // Initialize standard logger with prefix
    logger := log.New(log.Writer(), "DEBUG: ", log.LstdFlags|log.Lshortfile)

    ctx := context.Background()

    // Create two libp2p hosts
    logger.Println("Creating host1")
    host1, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
    if err != nil {
        logger.Fatalf("Failed to create host1: %v", err)
    }
    logger.Printf("Host1 created with ID: %s, Addresses: %v", host1.ID(), host1.Addrs())

    logger.Println("Creating host2")
    host2, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
    if err != nil {
        logger.Fatalf("Failed to create host2: %v", err)
    }
    logger.Printf("Host2 created with ID: %s, Addresses: %v", host2.ID(), host2.Addrs())
    defer host1.Close()
    defer host2.Close()

    // Initialize DHT for both hosts (default protocol is /ipfs/kad/1.0.0)
    logger.Println("Initializing DHT for host1")
    dht1, err := dht.New(ctx, host1, dht.Mode(dht.ModeServer))
    if err != nil {
        logger.Fatalf("Failed to initialize DHT for host1: %v", err)
    }
    logger.Println("Initializing DHT for host2")
    dht2, err := dht.New(ctx, host2, dht.Mode(dht.ModeClient))
    if err != nil {
        logger.Fatalf("Failed to initialize DHT for host2: %v", err)
    }

    // Bootstrap DHT
    logger.Println("Bootstrapping DHT for host1")
    if err := dht1.Bootstrap(ctx); err != nil {
        logger.Fatalf("Failed to bootstrap DHT for host1: %v", err)
    }
    logger.Println("Bootstrapping DHT for host2")
    if err := dht2.Bootstrap(ctx); err != nil {
        logger.Fatalf("Failed to bootstrap DHT for host2: %v", err)
    }

    // Connect the two hosts
    addr := host1.Addrs()[0].String() + "/p2p/" + host1.ID().String()
    logger.Printf("Connecting host2 to host1 at address: %s", addr)
    host2Addr, err := ma.NewMultiaddr(addr)
    if err != nil {
        logger.Fatalf("Failed to parse multiaddr: %v", err)
    }
    if err := host2.Connect(ctx, peer.AddrInfo{ID: host1.ID(), Addrs: []ma.Multiaddr{host2Addr}}); err != nil {
        logger.Fatalf("Failed to connect host2 to host1: %v", err)
    }
    logger.Println("Hosts connected successfully")

    // Verify connection stability
    logger.Println("Verifying connection from host2 to host1")
    if host2.Network().Connectedness(host1.ID()) != network.Connected {
        logger.Fatalf("Host2 not connected to host1")
    }
    logger.Println("Connection verified")

    // Wait for DHT to be ready
    logger.Println("Waiting for DHT routing tables to populate")
    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        for dht1.RoutingTable().Size() == 0 {
            logger.Println("Host1 routing table empty, waiting...")
            time.Sleep(3000 * time.Millisecond)
        }
        logger.Printf("Host1 routing table populated with %d peers", dht1.RoutingTable().Size())
    }()
    go func() {
        defer wg.Done()
        for dht2.RoutingTable().Size() == 0 {
            logger.Println("Host2 routing table empty, waiting...")
            time.Sleep(100 * time.Millisecond)
        }
        logger.Printf("Host2 routing table populated with %d peers", dht2.RoutingTable().Size())
    }()
    wg.Wait()

    // Store a key-value pair in the DHT from host2 with retry
    key := "/myapp/testkey"
    value := []byte("Hello, libp2p DHT!")
    logger.Printf("Storing key: %s, value: %s", key, value)
    const maxRetries = 3
    for attempt := 1; attempt <= maxRetries; attempt++ {
        logger.Printf("Attempt %d to store key-value in DHT", attempt)
        err = dht2.PutValue(ctx, key, value)
        if err == nil {
            logger.Println("Key-value pair stored successfully")
            break
        }
        logger.Printf("Failed to store key-value in DHT: %v", err)
        if attempt == maxRetries {
            logger.Fatalf("Failed to store key-value after %d attempts: %v", maxRetries, err)
        }
        time.Sleep(500 * time.Millisecond)
    }

    // Retrieve the value from the DHT using host1 with retry
    logger.Printf("Retrieving value for key: %s", key)
    ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second) // Increased timeout
    defer cancel()
    var retrievedValue []byte
    for attempt := 1; attempt <= maxRetries; attempt++ {
        logger.Printf("Attempt %d to retrieve value from DHT", attempt)
        retrievedValue, err = dht1.GetValue(ctxTimeout, key)
        if err == nil {
            logger.Printf("Retrieved value for key %s: %s", key, retrievedValue)
            break
        }
        logger.Printf("Failed to retrieve value from DHT: %v", err)
        if attempt == maxRetries {
            logger.Fatalf("Failed to retrieve value after %d attempts: %v", maxRetries, err)
        }
        time.Sleep(500 * time.Millisecond)
    }
    fmt.Printf("Retrieved value for key %s: %s\n", key, retrievedValue)
}
