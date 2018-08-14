package main

import (
	"net/http"
	"fmt"
	"net"
	"log"
	"strings"
	"encoding/json"
	"github.com/go-redis/redis"
	"runtime"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"database/sql"
	_"github.com/go-sql-driver/mysql"
	"math/rand"
)

type User struct {//mysql ile handshake yapmak için user modeli
	UserName string //`json:"username"`
	Password string//`json:"password"`
	SubKey string//`json:"subkey"`
	PubKey string//`json:"pubkey"`
}

type Client struct {//tcp'den bağlanan kullanıcının modeli
	conn net.Conn
	message string
}

type StatusMessage struct {//API response modeli
	Message string `json:"status_message"`
	StatusCode int `json:"status_code"`
}

type Publish struct {//API üzerinden bağlanan kullanıcının publish modeli
	Token string //`json:"token"`
	Message string //`json:"message"`
	Channel string //`json:"channel"`
	Pubkey string //`json:"pubkey"`
	Subkey string //`json:"subkey"`

}

type Subscribe struct {////API üzerinden bağlanan kullanıcının subscribe modeli
	Token string
	Subkey string
}

type Broker struct {//Server Send Event kapsamında kullanıcının tüm eventlerini içeren ana model
	Notifier chan []byte// Kullanıcının yaptığı tüm eventler burada değerlendiriliyor.
	newClients chan chan []byte// yeni client bağlanınca
	closingClients chan chan []byte// client disconnect olunca
	clients map[chan []byte]bool// tüm clientlere brodcast yayın yapmak için
	clientIpAdress []string//sisteme bağlı clientlerin ip adresleri
	clientTokens []string//sisteme bağlı clientlerin tokenleri
}


func NewServer() (broker *Broker) {//sistemi ayağa kaldırmak için gerekli instance
	broker = &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}
	go broker.listen()//sistem ayağa kalktığında eş zamanlı olarak bu metot çalışacak.
	return
}

func (broker *Broker) addNewClient(ipAddress string) (index int){//yeni client bağlanınca ip'si ekleniyor.
	broker.clientIpAdress =append(broker.clientIpAdress,ipAddress)
	index = len(broker.clientIpAdress)-1
	return

}

func (broker *Broker) findIpPosition(ipAddress string)  (pos int) {//verilen ip'nin dizideki pozisyonunu buluyor.
	for i,ip := range broker.clientIpAdress{
		if ip == ipAddress{
			pos = i
			return
		}
	}
	return
}

func (broker *Broker) closeClientToken(pos int){//kullanıcı çıktığında diziden çıkıyor.
	broker.clientTokens = append(broker.clientTokens[:pos], broker.clientTokens[pos+1:]...)
}

func (broker *Broker) closeClientIpAddres(pos int){//kullanıcı çıktığında
	broker.clientIpAdress = append(broker.clientIpAdress[:pos], broker.clientIpAdress[pos+1:]...)

}

func (broker *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {//client serverimiza bağlanıca..
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")//handshake için gerekli headerler
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	messageChan := make(chan []byte)
	broker.newClients <- messageChan//yeni client bağlanında devreye girecek olan channel
	newClientFullIp := req.RemoteAddr//client'in ip adresini alıyoruz.
	index := broker.addNewClient(newClientFullIp)//client'in dizideki indisini alıyoruz.
	token := broker.clientTokens[index]//ip indisini token dizisinde aynı  indise eşitliyoruz.
	client.SAdd("onlineUserTokens",token)//rediste uniq listemize  bu kullanıcının tokenını ekliyoruz.


	defer func() {//kullanıcı disconnect olduğunda bu channel notification olarak gidecek.
		broker.closingClients <- messageChan
	}()
	notify := rw.(http.CloseNotifier).CloseNotify()//3000 portuna bağlı client çıkış yapınca notify devreye giriyor..
	clientCloseFullIp := req.RemoteAddr// disconnect olan kullanıcının ip adresinin alıyoruz


	go func() {//user disconnect olunca eş zamanlı bunu channel ile notification olarak gönderecez.
		<-notify
		posClientIp := broker.findIpPosition(clientCloseFullIp)//disconnect olan kullanıcının dizideki indisini buluyoruz
		client.SRem("onlineUserTokens",broker.clientTokens[posClientIp])//yukarıda elde ettiğimiz indisteki tokeni bulıuyoruz ve bunu redisteki uniq listten çıkarıyoruz.
		broker.closeClientToken(posClientIp)//user'i token dizisinden çıkarıyoruz
		broker.closeClientIpAddres(posClientIp)//user'i ip dizisinden çıkarıyoruz.
		broker.closingClients <- messageChan//close notification'u gönderiyoruz.
	}()
	for {//burada ilgili tüm userlere sırasıyla ilgili projelerine broadcast mesaj göndericez.
		tokenPosition := broker.findIpPosition(newClientFullIp)//aktif userin token indisi
		token := broker.clientTokens[tokenPosition]//aktif user'in tokeni.
		data := ByteToStr(<-messageChan)//channel'den gelen mesajımız
		parsedData := strings.Split(data,":")//redisten gelen datayı parse ediyoruz(kanal ve mesaj olarak geliyor.)
		channels := strings.Split(parsedData[0],"_")//channel bilgisini elde ettik.
		isFindUserToken := client.SIsMember(channels[0][:4],token).Val()//userlerin ilgili projelerine ilgili mesajı ayırt etmek için rediste listede kontrol yapıyoruz.
		if isFindUserToken{
			fmt.Fprintf(rw, "data: %s\n\n", channels[1]+"_"+parsedData[1])
			flusher.Flush()
		}
	}

}

func (broker *Broker) listen() {//burada eş zamanlı olarak çalışan ilgili metotlarımızı tek bir noktadan yönetmek için select ifadesini kullanıyoruz.
	for {
		select {
		case s := <-broker.newClients: //yeni bir client bağlandı..

			broker.clients[s] = true

			onlieUsers := client.SMembers("onlineUserTokens")
			fmt.Println("online Users:",onlieUsers)
			onlineUsersCount := client.SCard("onlineUserTokens")
			fmt.Println("online users count:",onlineUsersCount)

		case s := <-broker.closingClients://Bir client ayrıldı ve mesaj göndermeyi bırakmak istiyoruz

			onlieUsers := client.SMembers("onlineUserTokens")
			fmt.Println("online Users:",onlieUsers)
			onlineUsersCount := client.SCard("onlineUserTokens")
			fmt.Println("online users count:",onlineUsersCount)
			delete(broker.clients, s)

		case event := <-broker.Notifier://sisteme bağlı tüm clientlere notify gönderiyoruz
			for clientMessageChan, _ := range broker.clients {

				clientMessageChan <- event

			}
		}
	}
}

func check(err error, message string) {//genel hata yönetimi mekanizması
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", message)
}

func JsonStatus(message string, status int, jw http.ResponseWriter)  {//Genel Response metodumuz
	jw.Header().Set("Content-Type", "application/json")
	return_message := &StatusMessage{Message: message, StatusCode: status}
	jw.WriteHeader(http.StatusCreated)
	json.NewEncoder(jw).Encode(return_message)
}

func ByteToStr(data []byte) string{//byte to str
	d :=string(data[8:])
	d = d[:len(d)-1]
	return d
}

func randToken() string {//random token üreten fonksiyon
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func handShake(w http.ResponseWriter, r *http.Request) {//API üzerinden gelen subscribe
	var u User
	_ = json.NewDecoder(r.Body).Decode(&u)

	if u.Password == ""  || u.UserName == "" || u.SubKey == "" || u.PubKey == "" {
		JsonStatus("Error! Required User name ,Password ,Subkey and Pubkey!" ,330, w)
		return
	}
	isCheck , token := checkUser(u)
	if isCheck == -1{
		JsonStatus("Error! Invalid User Info!" ,330, w)
		return
	}else{
		JsonStatus("Token:"+token ,200, w)
		return
	}
}

func checkUser(user User) (isCheck int ,token string) {//handchake yaparken gelen datamızı mysql ile kontrol ediyoruz
	isCheck = -1
	token =""
	c := user
	var userId,appId int
	stmtOut, err := db.Prepare("SELECT user_id  FROM account where user_name = ? AND  password = ?")
	check(err,"")
	err = stmtOut.QueryRow(c.UserName,c.Password).Scan(&userId)
	if userId > 0{
		stmtOut, err := db.Prepare("SELECT app_id  FROM app where user_id = ? AND  pub_key = ? AND sub_key = ?")
		check(err,"")
		err = stmtOut.QueryRow(userId,c.PubKey,c.SubKey).Scan(&appId)
		if appId > 0 {
			isCheck = appId
			token = randToken()
			client.SAdd(c.SubKey[:8],token)
		}
	}
	defer stmtOut.Close()
	return
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




func (broker *Broker)subscribeHttpToRedis(w http.ResponseWriter, r *http.Request) {//API üzerinden gelen subscribe
	var s Subscribe
	_ = json.NewDecoder(r.Body).Decode(&s)

	if s.Token == ""  || s.Subkey == "" {
		JsonStatus("Error! Required Token and Subkey Subkey!" ,330, w)
		return
	}

	stateSubkey := client.SIsMember("userSubkeys",s.Subkey[:5]).Val()//userin sistemde yarattığı proje check yapılmış mı diye kontrol yapılıyor
	if !stateSubkey{
		appId := checkSubKey(s.Subkey)//mysql check
		if appId > 0{
			client.SAdd("userSubkeys",s.Subkey[:5])//başarılı ise rediste check yapıldı olarak görünüyor.

		}else{
			fmt.Println("böyle bir subkey yok.")
			return
		}
	}
	s.Subkey =s.Subkey[:5]
	broker.clientTokens = append(broker.clientTokens,s.Token)
	client.SAdd(s.Subkey[:4],s.Token)

	isSubscribe,_ := client.Get(s.Subkey).Int64()
	if isSubscribe != 0 {
		JsonStatus("Active Subscribe!" ,321, w)
		return

	}else{
		JsonStatus("Successful!" ,200, w)
		client.Set(s.Subkey,1,0)

	}

	go func() {//user subscribe olduğunda artık redisten kendisini ilgilendiren mesaj geldiğinde onu dinleyecektir.

		pubsub := client.PSubscribe(s.Subkey+"_*")
		for{
			msg, err := pubsub.ReceiveMessage()
			if err != nil {
				panic(err)
			}
			chanRecieveRedisToHttp <- msg.String()//publih kanalımıza api'den gelen bilgileri ekledik
		}
	}()

}

func publishRedis(w http.ResponseWriter, r *http.Request) {//API üzerinden gelen publish
	var p Publish
	_ = json.NewDecoder(r.Body).Decode(&p)

	if p.Token == "" || p.Message == "" || p.Channel == "" || p.Subkey == "" || p.Pubkey == "" {

		JsonStatus("Required Pubkey and Subkey B" ,351,w)
		return
	}
	stateSubkey := client.SIsMember("userSubkeys",p.Subkey[:5]).Val()
	statePubkey := client.SIsMember("userPubkeys",p.Pubkey[:5]).Val()
	if !stateSubkey || !statePubkey {
		appId := checkPupKeySubKey(p.Pubkey,p.Subkey)//mysql check
		if appId > 0{
			client.SAdd("userSubkeys",p.Subkey[:5])//redis set
			client.SAdd("userPubkeys",p.Pubkey[:5])
		}else{
			fmt.Println("böyle bir pubkey subkey yok.")
			return
		}
	}

	p.Pubkey = p.Pubkey[:5]
	p.Subkey = p.Subkey[:5]

	arrayChannels := strings.Split(p.Channel,"|")
	arrayMesages := strings.Split(p.Message,"|")

	if len(arrayMesages) > len(arrayChannels){
		JsonStatus("channel sayisi mesaj sayisindan az olamaz" ,354, w)
		return
	}
	getToken := client.Get("token").Val()
	if p.Token ==getToken{

		JsonStatus("Data Published" ,200, w)
		go func() {
			for i,chann := range arrayChannels {//belirtilen çoklu kanallara çoklu mesaj gönderiliyor..
				p.Channel = p.Subkey + "_" + chann
				if i >= len(arrayMesages) {
					p.Message = ""
				} else {
					p.Message = arrayMesages[i]
				}
				chanSendHttpToRedisPublih <- p//Redis ile api arasında eş zamanlı dinlenen kanal
			}
		}()

	}else{
		JsonStatus("Error! Invalid Token !" ,355, w)
		return
	}
}




var chanSendHttpToRedisPublih = make(chan Publish)//apiPublish'den gelen datamızı publish için kanalımız
var chanRecieveRedisToHttp = make (chan string)//rediste kanala subscribe olur.
var db, err = sql.Open("mysql", "root:@/skyneb")

var client = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
})

func main(){

	check(err,"Mysql Connectted.")

	client.Set("token","tokenabc",0)
	client.Set("pubkey","pubkey123",0)

	var broker = NewServer()
	runtime.GOMAXPROCS(8)


	for {
		go func() {
			log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:3000", broker))
		}()


		go func() {
			for{
				msg := <- chanRecieveRedisToHttp
				broker.Notifier <- []byte(msg)
			}
		}()

		go func() {
			for{
				publih := <- chanSendHttpToRedisPublih//api'den gelenleri redise publish edioruz
				err = client.Publish(publih.Channel, publih.Message).Err()//kanal bilgisi api'den geliyor
				check(err,"publish edildi")
			}
		}()


		allowedHeaders := handlers.AllowedHeaders([]string{"X-Requested-With"})
		allowedOrigins := handlers.AllowedOrigins([]string{"*"})
		allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})

		r := mux.NewRouter()
		r.HandleFunc("/publish", publishRedis).Methods("POST")
		r.HandleFunc("/subscribe", broker.subscribeHttpToRedis).Methods("POST")
		r.HandleFunc("/handshake", handShake).Methods("POST")
		log.Fatal(http.ListenAndServe(":8000", handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods)(r)))
	}
}






