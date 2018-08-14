package main

import "net"
import (
    "fmt"
    "bytes"
    "encoding/gob"
    "time"
)
type publissh struct {//API üzerinden bağlanan kullanıcının publish modeli
    Token string //`json:"token"`
    Message string //`json:"message"`
    Channel string //`json:"channel"`
    Pubkey string //`json:"pubkey"`
    Subkey string //`json:"subkey"`
}

func main() {

  conn, err := net.Dial("tcp", ":8070")//serverun 8080 portuna bağlanır  206.189.62.46

  if err != nil {
    fmt.Println("Bağlantı Kurulamadı")
  }
  if conn != nil {
	fmt.Println("Bağlantı Kuruldu")
  }

  for  {

      msg := publissh{Token:"tokenawaw",Message:"selam ben tcp'den geliyorum | ben de geliyorum",Channel:"erdal | fatih",Pubkey:"858249ade867fb55dd7aa6b09cf83143",Subkey:"ca4264b16985de63e7d167a33661a4aa"}
      bin_buf := new(bytes.Buffer)
      gobobj := gob.NewEncoder(bin_buf)
      gobobj.Encode(msg)
      conn.Write(bin_buf.Bytes())
      time.Sleep(100 * time.Millisecond)

  }

}
