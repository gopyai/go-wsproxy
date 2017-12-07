package wsproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	DEBUG = false
)

func Handler(path, sockServer string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if DEBUG {
			fmt.Println("Incoming websocket request from:", r.RemoteAddr)
			fmt.Println("URI:", r.RequestURI)
			//fmt.Println("Method:", r.Method)
			//fmt.Println("Path:", r.URL.Path)
			//fmt.Println("Query:", r.URL.RawQuery)
		}

		upgrader := websocket.Upgrader{}
		wsClient, e := upgrader.Upgrade(w, r, nil)
		if e != nil {
			fmt.Println("Upgrade error")
			//http.Error(w, e.Error(), 400)
			return
		}
		defer wsClient.Close()

		wsServer, e := wsConnect("ws", sockServer, r.URL.Path[len(path):], r.URL.RawQuery)
		if e != nil {
			fmt.Println("Connect error")
			//http.Error(w, e.Error(), 400)
			return
		}
		defer wsServer.Close()

		wg := new(sync.WaitGroup)
		wg.Add(2)
		go wsProxy(wsClient, wsServer, wg)
		go wsProxy(wsServer, wsClient, wg)
		wg.Wait()
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
