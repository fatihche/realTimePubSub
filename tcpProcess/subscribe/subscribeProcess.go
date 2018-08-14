package main

import (
	"bufio"
	"fmt"
	"net"
	"runtime"
	"github.com/go-redis/redis"
	"bytes"
	"encoding/gob"
	_"github.com/go-sql-driver/mysql"
	"database/sql"
)

type Subscribe struct {////API üzerinden bağlanan kullanıcının subscribe modeli
	Token string
	Subkey string
}

func subscribeRedis(subscribee Subscribe) {//API üzerinden gelen subscribe
	subscribe :=subscribee

	if subscribe.Token == ""  || subscribe.Subkey == "" {
		fmt.Println("Error! Required Token and Subkey Subkey!")
		return
	}

	stateSubkey := client.SIsMember("userSubkeys",subscribe.Subkey[:5]).Val()
	if !stateSubkey{
		appId := checkSubKey(subscribe.Subkey)//mysql check
		if appId > 0{
			client.SAdd("userSubkeys",subscribe.Subkey[:5])

		}else{
			fmt.Println("böyle bir subkey yok.")//burada json dönücek ve bir hata kodu dönecek.
			return
		}
	}

	isSubscribe,_ := client.Get(subscribe.Subkey[5:10]).Int64()
	if isSubscribe != 0 {
		fmt.Println("Active Subscribe!")
		return

	}else{
		fmt.Println("Successful!")
		client.Set(subscribe.Subkey[5:10],1,0)

	}

	go func() {
		pubsub := client.PSubscribe(subscribe.Subkey[:5]+"_*")
		for{
			msg, err := pubsub.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			chanWriteRedisToSocket <- msg.String()//publih kanalımıza api'den gelen bilgileri ekledik
		}
	}()

}

func tcpClientConnect(conn net.Conn){
	buf := bufio.NewReader(conn) //clienti dinlemek için bir buffer oluşturduk
	tmp := make([]byte, 500)

	for {
		_,err := buf.Read(tmp)
		if err != nil {
			fmt.Printf("Client disconnected.\n")
			break
		}
		tmpbuff := bytes.NewBuffer(tmp)
		receiveSubscribe := new(Subscribe)
		gobobj := gob.NewDecoder(tmpbuff)
		gobobj.Decode(receiveSubscribe)
		publish  := *receiveSubscribe
		subscribeRedis(publish)
	}
}
func writeSocket(){//Sokete yazılacak ve soket redisten gelen mesjları okuyacak..
	for{
		data := <- chanWriteRedisToSocket
		fmt.Println("data:",data)
	}
}


func check(err error, message string) {
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}
func checkSubKey(subkey string) (isCheck int) {//handchake yaparken gelen datamızı mysql ile kontrol ediyoruz

	var appId int
	stmtOut, err := db.Prepare("SELECT app_id  FROM app where sub_key = ?")
	check(err,"")
	err = stmtOut.QueryRow(subkey).Scan(&appId)
	if appId > 0{
		isCheck =appId
	}else{
		isCheck = -1
	}

	defer stmtOut.Close()
	return
}

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
})
var db, err = sql.Open("mysql", "root:@/skyneb")
var chanWriteRedisToSocket = make(chan string)

func main() {
	runtime.GOMAXPROCS(8)
	ln, err := net.Listen("tcp", ":8080")
	client.Set("issubkey",0,0)
	check(err, "Server is ready.")

	for {
		conn, err := ln.Accept()
		check(err, "Accepted connection.")

		go tcpClientConnect(conn)
		go writeSocket()
	}
}