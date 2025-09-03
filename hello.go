package main

import (
    "context"
    "fmt"
    "log"
    "time"

    dht "github.com/libp2p/go-libp2p-kad-dht"
    libp2p "github.com/libp2p/go-libp2p"
)

func makeNode() (*dht.IpfsDHT, error) {
    ctx := context.Background()

    host, err := libp2p.New()
    if err != nil {
        return nil, fmt.Errorf("failed to create libp2p host: %w", err)
    }

    // Create a new DHT instance
    kdht, err := dht.New(ctx, host)
    if err != nil {
        return nil, fmt.Errorf("failed to create DHT: %w", err)
    }

    // Bootstrap the DHT
    if err := kdht.Bootstrap(ctx); err != nil {
        return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
    }

    return kdht, nil
}

func put(kdht *dht.IpfsDHT, key string, value []byte) {
    ctx := context.Background()
    // Use a valid DHT key prefix (e.g., "/appname/") for storing values
    err := kdht.PutValue(ctx, "/myapp/"+key, value)
    if err != nil {
        log.Printf("PutValue error: %v", err)
    } else {
        fmt.Printf("Successfully stored key: %s\n", key)
    }
}

func get(kdht *dht.IpfsDHT, key string) []byte {
    ctx := context.Background()
    // Use the same key format for retrieval
    ch, err := kdht.SearchValue(ctx, "/myapp/"+key)
    if err != nil {
        log.Printf("SearchValue error: %v", err)
        return nil
    }

    for val := range ch {
        fmt.Printf("Found value for key=%s: %s\n", key, string(val))
        return val
    }

    fmt.Printf("No value found for key=%s\n", key)
    return nil
}

func main() {
    kdht, err := makeNode()
    if err != nil {
        log.Fatalf("Failed to start node: %v", err)
    }

    // Let the DHT routing table populate
    time.Sleep(30 * time.Second)

    fmt.Printf("Routing table size: %d\n", kdht.RoutingTable().Size())

    // Store a value
    put(kdht, "foo", []byte("bar"))

    // Small wait to simulate network propagation
    time.Sleep(1 * time.Second)

    // Retrieve the value
    val := get(kdht, "foo")
    fmt.Printf("Retrieved: %s\n", string(val))
}

