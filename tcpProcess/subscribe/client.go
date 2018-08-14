package main

import "net"
import (
    "fmt"
    "bytes"
    "encoding/gob"
    "time"
    "bufio"
    "os"
)
type subscribe struct {////API üzerinden bağlanan kullanıcının subscribe modeli
    Token string
    Subkey string
}

func main() {

  conn, err := net.Dial("tcp", ":8080")//serverun 8080 portuna bağlanır  206.189.62.46

  if err != nil {
    fmt.Println("Bağlantı Kurulamadı")

  }
  if conn != nil {
	fmt.Println("Bağlantı Kuruldu")
  }

  for  {

      reader := bufio.NewReader(os.Stdin)//yazdığımız değeri okuyoruz
      fmt.Print("Data Send:")
      text, _ := reader.ReadString('\n')//yazdığımız değeri okuyoruz
      fmt.Println("text:",text)

      sb := subscribe{Token:"tokenawaw",Subkey:"ca4264b16985de63e7d167a33661a4aa"}
      bin_buf := new(bytes.Buffer)
      gobobj := gob.NewEncoder(bin_buf)
      gobobj.Encode(sb)
      conn.Write(bin_buf.Bytes())
      time.Sleep(1000 * time.Millisecond)

  }

}
