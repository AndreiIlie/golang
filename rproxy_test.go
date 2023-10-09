package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckIP(t *testing.T) {
    // Test case 1: IP not in map
    ip := "192.168.0.1"
    expected := true
    actual := checkIP(ip)
    if actual != expected {
        t.Errorf("checkIP(%s) = %v; expected %v", ip, actual, expected)
    }

    // Test case 2: IP in map, but expired
    ip = "192.168.0.2"
    ipMap[ip] = &ipData{last: time.Now().Add(-time.Duration(interval+1) * time.Second), count: 5}
    expected = true
    actual = checkIP(ip)
    if actual != expected {
        t.Errorf("checkIP(%s) = %v; expected %v", ip, actual, expected)
    }

    // Test case 3: IP in map, not expired, but over max requests
    ip = "192.168.0.3"
    ipMap[ip] = &ipData{last: time.Now(), count: maxRequests}
    expected = false
    actual = checkIP(ip)
    if actual != expected {
        t.Errorf("checkIP(%s) = %v; expected %v", ip, actual, expected)
    }

    // Test case 4: IP in map, not expired, and under max requests
    ip = "192.168.0.4"
    ipMap[ip] = &ipData{last: time.Now(), count: maxRequests - 1}
    expected = true
    actual = checkIP(ip)
    if actual != expected {
        t.Errorf("checkIP(%s) = %v; expected %v", ip, actual, expected)
    }
}

func TestAlterData(t *testing.T) {
    // Test case 1: empty map
    data := make(map[string]interface{})
    alterData(data)
    if data["foo"] != "bar" {
        t.Errorf("alterData did not set foo to bar")
    }

    // Test case 2: map with existing key
    data = map[string]interface{}{"foo": "baz"}
    alterData(data)
    if data["foo"] != "bar" {
        t.Errorf("alterData did not overwrite foo with bar")
    }
}

func TestProcessRequest(t *testing.T) {
    // Test case 1: valid request
    req, err := http.NewRequest("GET", "/todos/1", nil)
    if err != nil {
        t.Fatal(err)
    }
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(processRequest)
    handler.ServeHTTP(rr, req)
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }
    expectedContentType := "application/json; charset=utf-8"
    if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
        t.Errorf("handler returned unexpected content type: got %v want %v", contentType, expectedContentType)
    }
    var todo struct {
        UserID int    `json:"userId"`
        ID     int    `json:"id"`
        Title  string `json:"title"`
        Completed bool `json:"completed"`
    }
    if err := json.NewDecoder(rr.Body).Decode(&todo); err != nil {
        t.Errorf("handler returned invalid JSON: %v", err)
    }
    if todo.UserID != 1 {
        t.Errorf("handler returned unexpected userID: got %v want %v", todo.UserID, 1)
    }
    if todo.ID != 1 {
        t.Errorf("handler returned unexpected ID: got %v want %v", todo.ID, 1)
    }
    if todo.Title != "delectus aut autem" {
        t.Errorf("handler returned unexpected title: got %v want %v", todo.Title, "sunt aut facere repellat provident occaecati excepturi optio reprehenderit")
    }
    if todo.Completed != false {
        t.Errorf("handler returned unexpected completed: got %v want %v", todo.Completed, false)
    }

    // Test case 2: too many requests
    req, err = http.NewRequest("GET", "/posts/1", nil)
    if err != nil {
        t.Fatal(err)
    }
    ip := "192.168.0.1"
    ipMap[ip] = &ipData{last: time.Now(), count: maxRequests}
    req.RemoteAddr = ip
    rr = httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    if status := rr.Code; status != http.StatusTooManyRequests {
        t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusTooManyRequests)
    }
}
