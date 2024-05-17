package main

import "net"

type User struct {
	server *Server
	C      chan string
	conn   net.Conn
	Name   string
	Addr   string
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		server: server,
		C:      make(chan string),
		conn:   conn,
		Name:   userAddr,
		Addr:   userAddr,
	}
	go user.ListenMessage()
	return user
}

// 用户上线的业务
func (user *User) Online() {
	// 用户上线
	user.server.MapLock.Lock()
	user.server.OnlineMap[user.Name] = user
	user.server.MapLock.Unlock()
	// 用户上线消息广播
	user.server.BoradCast(user, "已上线")
}

// 用户下线逻辑
func (user *User) Offine() {
	user.server.MapLock.Lock()
	delete(user.server.OnlineMap, user.Name)
	user.server.MapLock.Unlock()
	user.server.BoradCast(user, "下线")
}


func (user *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户都有哪些

		user.server.MapLock.Lock()
		for _, user := range user.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + "在线...\n"
			user.conn.
		}
		user.server.MapLock.Unlock()
	} else {
		user.server.BoradCast(user, msg)
	}
}

// 监听当前user channel ，一旦有消息就直接发送给对端客户端
func (user *User) ListenMessage() {
	for {
		msg := <-user.C
		user.conn.Write([]byte(msg + "\n"))
	}
}
