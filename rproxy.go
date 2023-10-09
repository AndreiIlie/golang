package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
    maxRequests = 10
    interval    = 60
)

type ipData struct {
    count int
    last  time.Time
}

type Request struct {
    ip   string `bson:"_id"`
    path string `bson:"path"`
}

type Response struct {
    data map[string]interface{} `bson:"data"`
    err  error                  `bson:"err"`
}

var (
    ipMap = make(map[string]*ipData)
    mutex sync.Mutex
)

func checkIP(ip string) bool {
    mutex.Lock()
    defer mutex.Unlock()
    ipMapData, ok := ipMap[ip]
    if !ok {
        return true
    }
    now := time.Now()
    if now.Sub(ipMapData.last) >= time.Duration(interval)*time.Second {
        ipMapData.count = 0
        return true
    }
    if ipMapData.count >= maxRequests {
        return false
    }
    return true
}

func registerIP(ip string) {
    mutex.Lock()
    defer mutex.Unlock()
    ipMapData, ok := ipMap[ip]
    if !ok {
        ipMapData = &ipData{}
        ipMap[ip] = ipMapData
    }
    now := time.Now()
    if now.Sub(ipMapData.last) >= time.Duration(interval)*time.Second {
        ipMapData.count = 0
    }
    ipMapData.count++
    ipMapData.last = now

}

func alterData(data map[string]interface{}) {
    data["foo"] = "bar"
}

func processRequest(w http.ResponseWriter, r *http.Request) {
    ip := r.RemoteAddr
    if !checkIP(ip) {
        http.Error(w, "Too many requests", http.StatusTooManyRequests)
        return
    }
    registerIP(ip)

    path := r.URL.Path

    resp, err := http.Get("http://jsonplaceholder.typicode.com" + path)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    contentType := resp.Header.Get("Content-Type")
    if contentType != "application/json; charset=utf-8" {
        http.Error(w, "Unexpected content-type", http.StatusInternalServerError)
        return
    }
    var data map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    alterData(data)

    // Insert request and response to MongoDB
    clientOptions := options.Client().ApplyURI("mongodb+srv://<username>:<password>@<cluster>/?retryWrites=true&w=majority")
    client, err := mongo.Connect(context.TODO(), clientOptions)
    if err == nil {
        defer client.Disconnect(context.TODO())

        collection := client.Database("rproxy").Collection("requests")
        request := Request{ip: ip, path: path}
        log.Printf("Request: %+v\n", request)

        _, err = collection.InsertOne(context.TODO(), request)
        if err != nil {
            log.Fatal(err)
        }

        collection = client.Database("rproxy").Collection("responses")
        response := Response{data: data, err: err}
        _, err = collection.InsertOne(context.TODO(), response)
        if err != nil {
            log.Fatal(err)
        }
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    err = json.NewEncoder(w).Encode(data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func main() {
    http.HandleFunc("/", processRequest)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
