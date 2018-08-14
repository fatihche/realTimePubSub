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
	"strings"
)

type Publish struct {//API üzerinden bağlanan kullanıcının publish modeli
	Token string //`json:"token"`
	Message string //`json:"message"`
	Channel string //`json:"channel"`
	Pubkey string //`json:"pubkey"`
	Subkey string //`json:"subkey"`

}
func publishChannelRedis(publishh Publish) {
	publish :=publishh
	if publish.Token == "" || publish.Message == "" || publish.Channel == "" || publish.Subkey == "" || publish.Pubkey == "" {
		fmt.Println("boş bilgiler var.")
		return
	}

	stateSubkey := client.SIsMember("userSubkeys",publish.Subkey[:5]).Val()
	statePubkey := client.SIsMember("userPubkeys",publish.Pubkey[:5]).Val()
	if !stateSubkey || !statePubkey {
		appId := checkPupKeySubKey(publish.Pubkey,publish.Subkey)//mysql check
		if appId > 0{
			fmt.Println("app id:",appId)
			client.SAdd("userSubkeys",publish.Subkey[:5])//redis set
			client.SAdd("userPubkeys",publish.Pubkey[:5])
		}else{
			fmt.Println("böyle bir pubkey subkey yok.")//burada json dönücek ve bir hata kodu dönecek.
			return
		}
	}

	arrayChannels := strings.Split(publish.Channel,"|")
	arrayMesages := strings.Split(publish.Message,"|")

	if len(arrayMesages) > len(arrayChannels){
		fmt.Println("channel sayisi messaj sayisindan az olamaz")
		return

	}
		go func() {

			for i,chann := range arrayChannels {//belirtilen çoklu kanallara çoklu mesaj gönderiliyor..
				publish.Channel = publish.Subkey[:5] + "_" + chann
				if i >= len(arrayMesages) {
					publish.Message = ""
				} else {
					publish.Message = arrayMesages[i]
				}
				chanSendTcpToRedisPublish <- publish//Redis ile api arasında eş zamanlı dinlenen kanal
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
		//encode decode
		tmpbuff := bytes.NewBuffer(tmp)
		receivePublish := new(Publish)
		gobobj := gob.NewDecoder(tmpbuff)
		gobobj.Decode(receivePublish)
		publish  := *receivePublish
		publishChannelRedis(publish)
	}
}
func publishRedis(){//redise userden gelen mesjları publish ediyoruz.
	for{
		data := <- chanSendTcpToRedisPublish
		err := client.Publish(data.Channel, data.Message).Err()//kanal bilgisi api'den geliyor
		check(err,"publish edildi")
	}
}


func check(err error, message string) {
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}

func checkPupKeySubKey(pubkey string,subkey string) (isCheck int) {//handchake yaparken gelen datamızı mysql ile kontrol ediyoruz

	var appId int
	stmtOut, err := db.Prepare("SELECT app_id  FROM app where pub_key = ? AND  sub_key = ?")
	check(err,"")
	err = stmtOut.QueryRow(pubkey,subkey).Scan(&appId)
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
var chanSendTcpToRedisPublish = make(chan Publish)

func main() {
	runtime.GOMAXPROCS(8)
	ln, err := net.Listen("tcp", ":8070")
	check(err, "Server is ready.")

		for {
			conn, err := ln.Accept()
			check(err, "Accepted connection.")

			go tcpClientConnect(conn)
			go publishRedis()
		}
}