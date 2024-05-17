package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type Server struct {
	OnlineMap map[string]*User
	// 消息广播的Channel
	Message chan string
	Ip      string
	Port    int
	MapLock sync.RWMutex
}

// 监听Message广播消息的goroutine，一旦有消息，就发送

func (server *Server) ListenMessage() {
	for {
		msg := <-server.Message
		server.MapLock.Lock()
		for _, cli := range server.OnlineMap {
			cli.C <- msg
		}
		server.MapLock.Unlock()
	}
}

func (server *Server) BoradCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	server.Message <- sendMsg
}

func (server *Server) Handler(conn net.Conn) {
	fmt.Println("链接建立成功")
	// 当前用户上线,将用户加入OnlineMap中
	user := NewUser(conn, server)
	user.Online()

	go func() {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)
		if n == 0 {
			user.Offine()
			return
		}
		if err != nil && err != io.EOF {
			fmt.Println("Conn Read err: ", err)
			return
		}
		// 提取用户的消息，去除'\n'
		msg := string(buf[:n-1])
		// 用户针对Message消息进行处理
		user.DoMessage(msg)
	}()
	select {}
}

// 创建一个server的接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
		Ip:        ip,
		Port:      port,
	}
	return server
}

// 启动服务器的方法

func (server *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Ip, server.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	// close listen socket
	defer listener.Close()
	// 启动监听Message
	go server.ListenMessage()
	// accept
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
		// do handle
		go server.Handler(conn)
	}
}
