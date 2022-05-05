package main

import (
	"server"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strconv"
	"time"
)
//三方库和本地库
//https://blog.csdn.net/woshiyuanlei/article/details/115456598
//开启方法
func runmethod(c *gin.Context) {
	port := httpGetParams("port",c)
	deadtime := httpGetDefaultParams("deadtime","2",c)
	//输出json结果给调用方
	if (port!=""&& deadtime !=""){

		deadtime1,err:=strconv.Atoi(deadtime)
		if(err!=nil){
			c.JSON(http.StatusBadRequest, gin.H{
				"message":  "deadtime参数必须为Int",
			})
			return
		}else if(deadtime1<=0){
			c.JSON(http.StatusBadRequest, gin.H{
				"message":  "deadtime参数必须为正整数",
			})
			return
		}else if(deadtime1>24){
			c.JSON(http.StatusBadRequest, gin.H{
				"message":  "deadtime参数必须小于等于24",
			})
			return
		}
		token:=server.PStartProxyDebug(port,false)
		fmt.Println("token:",token)
		if token!=""{
			//t_duation,_:=time.ParseDuration(deadtime)
			go autoClose(deadtime1,port,token)

			c.JSON(http.StatusOK, gin.H{
				"message":  "端口打开成功",
				"token":token,
				"info":"端口将于"+deadtime+"hours后关闭",
			})

		}else{
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":  "端口打开失败",
			})
		}
	}else{
		c.JSON(http.StatusBadRequest, gin.H{
			"message":  "port参数没有传递",
		})
	}
}
//自动关闭方法
func autoClose(deadtime1 int,port ,token string)  {
	//测试
	fmt.Println("等待关闭")
	//time.Sleep(time.Duration(deadtime1)*time.Second)
	//生产
	time.Sleep(time.Duration(deadtime1)*time.Hour)
	fmt.Println("关闭")
	server.PStopProxy(port+token)
}
//关闭方法
func stopmethod(c *gin.Context) {
	port := httpGetParams("port",c)
	token := httpGetParams("token",c)
	//输出json结果给调用方
	if port!=""&&token!=""{
		if server.PStopProxy(port+token){
			c.JSON(http.StatusOK, gin.H{
				"message":  "端口关闭成功",
			})
		}else{
			c.JSON(http.StatusInternalServerError, gin.H{
				"message":  "端口关闭失败",
			})
		}
	}else{
		c.JSON(http.StatusBadRequest, gin.H{
			"message":  "port或token参数没有传递",
		})
	}
}


func httpGetParams(key string,c *gin.Context) string{
	ret := c.Query(key)
	if ret ==""{
		ret = c.PostForm(key)
		if ret ==""{
			fmt.Println("获取不到参数")
		}else{
			fmt.Println("PostForm获取到参数:"+ret)
		}
	}else{
		fmt.Println("Query获取到参数:"+ret)
	}
	return ret
}
func httpGetDefaultParams(key,Default string,c *gin.Context) string{
	ret := c.DefaultQuery(key,Default)
	if ret ==""{
		ret =c.DefaultPostForm(key,Default)
		if ret ==""{
			fmt.Println("获取不到参数")
		}else{
			fmt.Println("PostForm获取到参数:"+ret)
		}
	}else{
		fmt.Println("Query获取到参数:"+ret)
	}
	return ret
}
func main() {
	r := gin.Default()
	defer func() {
		err:=recover()
		if err !=nil{
			fmt.Println(err)
			fmt.Println("缺少参数使用8888端口进行代理")
			err=r.Run(":8888")
			if err !=nil{
				fmt.Println(err)
				os.Exit(1)
			}else{
				fmt.Println("成功开启代理端口",8888)
			}
		}
	}()
	r.GET("/WebController/run", runmethod);
	r.POST("/WebController/run", runmethod);
	r.GET("/WebController/stop", stopmethod);
	r.POST("/WebController/stop", stopmethod);
	r.Run(":"+os.Args[1])
}
