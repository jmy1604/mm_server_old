package main

import (
	"fmt"
	"io"
	"net"
	"time"
)

type ReflectConn struct {
	conn *net.TCPConn
}

func (this *ReflectConn) work_loop() {
	rb := make([]byte, 4096)
	for {

		rl, err := this.conn.Read(rb)

		if io.EOF == err {
			fmt.Println("[%v]另一端关闭了连接", this.conn.RemoteAddr().String())
			return
		}

		if nil != err {
			return
		}

		if rl > 0 {
			fmt.Println("收到了消息", string(rb[:rl]))
			this.conn.Write(rb[:rl])
		}

		time.Sleep(time.Millisecond * 200)
	}
}

func new_reflectconn(conn *net.TCPConn) *ReflectConn {
	c_ret := &ReflectConn{}
	c_ret.conn = conn

	return c_ret
}

// ------------------------------------------------------

func main() {
	addr, err := net.ResolveTCPAddr("tcp", "192.168.1.16:8080")
	if err != nil {
		fmt.Println("net.ResolveTCPAddr ", err.Error())
		return
	}

	listener, err := net.ListenTCP(addr.Network(), addr)
	if err != nil {
		return
	}

	var conn *net.TCPConn

	fmt.Printf("[%v]服务器监听中\n", addr)

	for {
		fmt.Printf("[%v]等待新的连接", addr)
		conn, err = listener.AcceptTCP()

		if err != nil {
			fmt.Printf("[%v] Accept Error !\n", addr)
		}

		fmt.Printf("[%v]客户端连接成功 %v", addr, conn.RemoteAddr())

		conn.SetKeepAlive(true)
		conn.SetKeepAlivePeriod(time.Minute * 2)

		tmp_re_conn := new_reflectconn(conn)
		go tmp_re_conn.work_loop()
	}

	return
}
