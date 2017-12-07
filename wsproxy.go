package wsproxy

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	DEBUG = false

	sErrOpenOverLimit = "Too many opened connection"
	errOpenLimitValue = errors.New("Valid limit value must is limit >= 0")
)

func Handler(path, sockServer string, openLimit int) http.HandlerFunc {
	if openLimit < 0 {
		panic(errOpenLimitValue)
	}
	var lck struct {
		sync.Mutex
		cnt int
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Open limit checking
		// Increment number of connection
		lck.Lock()
		if lck.cnt >= openLimit {
			lck.Unlock()
			http.Error(w, sErrOpenOverLimit, http.StatusTooManyRequests)
			return
		}
		lck.cnt++
		lck.Unlock()

		// Debug message
		if DEBUG {
			fmt.Println("Incoming websocket request from:", r.RemoteAddr)
			fmt.Println("URI:", r.RequestURI)
			//fmt.Println("Method:", r.Method)
			//fmt.Println("Path:", r.URL.Path)
			//fmt.Println("Query:", r.URL.RawQuery)
		}

		// Upgrade standard HTTP connection into websocket
		upgrader := websocket.Upgrader{}
		wsClient, e := upgrader.Upgrade(w, r, nil)
		if e != nil {
			fmt.Println("Upgrade error")
			//http.Error(w, e.Error(), 400)
			return
		}
		defer wsClient.Close()

		// Connect to websocket server
		wsServer, e := wsConnect("ws", sockServer, r.URL.Path[len(path):], r.URL.RawQuery)
		if e != nil {
			fmt.Println("Connect error")
			//http.Error(w, e.Error(), 400)
			return
		}
		defer wsServer.Close()

		// Proxy between server and client
		wg := new(sync.WaitGroup)
		wg.Add(2)
		go wsProxy(wsClient, wsServer, wg)
		go wsProxy(wsServer, wsClient, wg)
		wg.Wait()

		// Open limit checking
		// Decrement number of connection
		lck.Lock()
		lck.cnt--
		lck.Unlock()
	})
}

func wsConnect(scheme, hostPort, path, rawQuery string) (*websocket.Conn, error) {
	dialer := websocket.DefaultDialer

	u := url.URL{Scheme: scheme, Host: hostPort, Path: path, RawQuery: rawQuery}
	urlStr := u.String()
	if DEBUG {
		fmt.Println("Client connect to:", urlStr)
	}
	c, _, e := dialer.Dial(urlStr, nil)
	if e != nil {
		return nil, e
	}
	return c, nil
}

func wsProxy(readFrom, writeTo *websocket.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		msgType, msg, e := readFrom.ReadMessage()
		if e != nil {
			switch e.(type) {
			case *websocket.CloseError:
				//fmt.Println("Receive close frame")
				wsWriteCloseMessage(writeTo)
				return
			}
			if DEBUG {
				fmt.Println("Error type:", reflect.TypeOf(e), "error:", e)
			}
			return
		}
		if e := writeTo.WriteMessage(msgType, msg); e != nil {
			if DEBUG {
				fmt.Println("Error type:", reflect.TypeOf(e), "error:", e)
			}
			return
		}
	}
}

func wsWriteCloseMessage(c *websocket.Conn) {
	c.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}
