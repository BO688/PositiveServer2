package server

import (
	"bufio"
	"fmt"
	"github.com/go-basic/uuid"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)
//定时器解决不及时关闭
//关闭需要开启后返回的uuid作为凭证
//统计所有可用端口范围


//代理端口集合
var PortMap =make(map[string]net.Listener,65536)
var debug=true
//开启代理端口
func PStartProxy( Port string) string{
	//代理服务端口
	Listen,err:=net.Listen("tcp","0.0.0.0:"+Port)
	if err !=nil{
		fmt.Println("代理端口失败")
		return ""
	}else{
		fmt.Println("成功开启代理端口",Port)
		go sub(Listen)
		//生成端口token
		token:=uuid.New()
		PortMap[Port+token]=Listen
		return token
	}
}
func PStartProxyDebug( Port string,debug1 bool) string{
	debug=debug1
	return PStartProxy(Port)
}
//关闭代理端口
func PStopProxy( Port string) bool{
	if PortMap[Port] ==nil{
		return false
	}else{
		PortMap[Port].Close()
		delete(PortMap,Port)
		return true
	}
}
//代理子携程
func sub(listen net.Listener){
	for   {
		conn,err:=listen.Accept()
		if err==nil{
			go mid_channel(conn)
		}

	}
}

//获取需要代理的连接A，并且建立新的连接B，随后对A获取写入B,获取B写入A
func mid_channel(conn net.Conn)  {
	reader := bufio.NewReader(conn)
	//请求方法，以此判断https与http
	var method string
	//请求地址
	var address string
	//请求内容
	var total string
	//异常
	var err error
	//每行数据
	var msg string
	//i:=0
	//flag:=false
	defer func() {
		re:=recover()
		if re!=nil &&debug{
			fmt.Println(re)
		}
	}()
	msg,err=reader.ReadString('\n')
	//第一行报文
	str:= strings.Split(msg, " ")
	//获取地址和方法
	address=str[1]
	method=str[0]
	//http连接没有这个Connect请求方式
	if method!="CONNECT" {
		//http方式
		total=msg
		for ;err==nil;{
			//获取每行报文
			//设置超时否则一直阻塞
			conn.SetReadDeadline(time.Now().Add(time.Second*1))
			if strings.HasPrefix(msg,"Host: "){
				address=strings.Replace(msg,"Host: ","",1)
				address=strings.Replace(address,"\n","",2)
				address=strings.Replace(address,"\r","",2)
			}
			msg,err=reader.ReadString('\n')
			if err==nil {
				total+=msg
			}
		}
		if debug{
			fmt.Println(string([]byte{27, 91, 57, 49, 109}), "HTTP front-------:"+total, string([]byte{27, 91, 48, 109}))

		}
		HTTPConnect(address,total,conn)
	} else{
		//HTTPS方式

		for  ;err==nil;{
			//获取每行报文
			//https的请求报文不能包含Proxy-Connection
			if(!strings.HasPrefix(msg,"Proxy-Connection")){
				total+=msg
			}
			//判定为https
			if(strings.HasPrefix(msg,"Host:")){
				break
			}
			msg,err=reader.ReadString('\n')
		}
		if debug{
			fmt.Println(string([]byte{27, 91, 57, 49, 109}), "HTTPS front-------:"+total, string([]byte{27, 91, 48, 109}))

		}
		HTTPSConnect(address,total,conn)
	}

}
//建立连接后会发送一次读取一次随后关闭
func HTTPConnect(address ,msg string,connF net.Conn)  {
	connT,err:=net.Dial("tcp", address)

	if(err!=nil){
		//有些没有带端口就是默认80，需要加上
		connT,err=net.Dial("tcp", address+":80")
		if(err!=nil){
			if(debug){fmt.Println(string([]byte{27, 91, 57, 53, 109}),"HTTP",address,err.Error(),string([]byte{27, 91, 48, 109}))}
			connF.Close()
		}
	}
	if(err==nil){
		//connF.SetDeadline( time.Now().Add(time.Second*5))
		//connT.SetDeadline( time.Now().Add(time.Second*5))
		if(debug){
			fmt.Println(string([]byte{27, 91, 57, 53, 109}),"to first target HTTP:\n",msg,string([]byte{27, 91, 48, 109}))
		}
		//写入请求报文
		connT.Write([]byte(msg))
		//设置超时
		connT.SetReadDeadline( time.Now().Add(time.Second*5))
		connF.SetReadDeadline(time.Now().Add(time.Second*5))

		var wg sync.WaitGroup
		wg.Add(2)
		//堵塞读取source，写入target线程
		go Channels(connT,connF,&wg)
		//堵塞读取target，写入source线程
		 Channels(connF,connT,&wg)
		//	主线程接手任务

	}


}


//建立起连接后会一直复用这两个链接
func HTTPSConnect(address string,msg string,connF net.Conn){
	connT,err:=net.Dial("tcp",address)
	if(err!=nil){
		//一般为443带有端口号，不需要再考虑省略80的情况
		fmt.Println(string([]byte{27, 91, 57, 52, 109}),"HTTPS",address,err.Error(),string([]byte{27, 91, 48, 109}))
		connF.Close()
	}else{
		if(debug) {
			fmt.Println(string([]byte{27, 91, 57, 52, 109}), "to first target HTTPS:\n", msg, string([]byte{27, 91, 48, 109}))
		}
		//应答https的建立
		connF.Write([]byte(msg))
		//需要用到锁来维持两个链接同时开启，同时关闭

		var wg sync.WaitGroup
		wg.Add(2)
		//堵塞读取source，写入target线程
		go Channels(connT,connF,&wg)
		//堵塞读取target，写入source线程
		 Channels(connF,connT,&wg)
		//	主线程接手任务

	}
}
//必须要加锁，因为类似TCP挥手那样保持数据正常传输，双方结束才可以断开
func Channels(source ,target net.Conn, wg1 *sync.WaitGroup)  {
	defer func() {
		recover()
		wg1.Done()
		if(debug){fmt.Println("等待关闭",source.LocalAddr(),source.RemoteAddr())}
		wg1.Wait()
		source.Close()
		if(debug){fmt.Println("已经关闭",source.LocalAddr(),source.RemoteAddr())}
	}()
	for{
		buf := [512]byte{}
		n, err := source.Read(buf[:])
		target.Write(buf[:n])
		if err == io.EOF {
			break
		}else if err !=nil {
			if(debug){fmt.Println("target recv failed, err:", err)}
			return
		}else{
			source.SetReadDeadline(time.Now().Add(time.Second*5))
		}
		//fmt.Println(string(buf[:n]))
	}
}

//优化：
// HTTP连接需要传输时长，是否会在传输文件大的时候因为设置超时时间断开呢？
//可以在for循环读取数据的时候进行重复设置超时时间
