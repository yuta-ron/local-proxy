package main

import (
	"io"
	"log"
	"net"
	"net/http"
)

// ハンドラーの定義
func handleHTTPProxy(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received request for: %s\n", r.URL.String())

    // HTTP CONNECT メソッドを処理
    if r.Method == http.MethodConnect {
        handleHTTPS(w, r)
    } else {
        // 通常のHTTPリクエストの処理
        proxy := &http.Transport{}
        proxyRequest, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        proxyRequest.Header = r.Header
        response, err := proxy.RoundTrip(proxyRequest)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadGateway)
            return
        }
        defer response.Body.Close()
        copyHeader(w.Header(), response.Header)
        w.WriteHeader(response.StatusCode)
        io.Copy(w, response.Body)
    }
}

// HTTPSトンネルを処理する関数
func handleHTTPS(w http.ResponseWriter, r *http.Request) {
    destConn, err := net.Dial("tcp", r.Host)
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
    hijacker, ok := w.(http.Hijacker)
    if !ok {
        http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
        return
    }
    clientConn, _, err := hijacker.Hijack()
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }

    go transfer(destConn, clientConn)
    go transfer(clientConn, destConn)
}

// データの転送を行う関数
func transfer(destination io.WriteCloser, source io.ReadCloser) {
    defer destination.Close()
    defer source.Close()
    io.Copy(destination, source)
}

// ヘッダーのコピー
func copyHeader(dst, src http.Header) {
    for k, vv := range src {
        for _, v := range vv {
            dst.Add(k, v)
        }
    }
}

func main() {
    proxy := &http.Server{
        Addr:    "127.0.0.1:8080", // プロキシのリッスンアドレスとポート
        Handler: http.HandlerFunc(handleHTTPProxy),
    }

    log.Println("Starting proxy server on :8080")
    if err := proxy.ListenAndServe(); err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}