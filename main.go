package main

import (
	"encoding/json"
	"fmt"
	_ "log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/cors"
	"github.com/spacemonkeygo/spacelog"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/weave-lab/flanders/db"
	"github.com/weave-lab/flanders/hep"

	// Choose your db handler or import your own here
	// _ "lab.getweave.com/weave/flanders/db/influx"
	_ "github.com/weave-lab/flanders/db/mongo"
)

func main() {
	logger := spacelog.GetLogger()
	logger.Debug("Testing logger")
	if logger.DebugEnabled() {
		fmt.Print("ENABLED!!!!")
	}
	go UDPServer("0.0.0.0", 9060)
	WebServer("0.0.0.0", 8080)
	// quit := make(chan struct{})
	// <-quit
}

var test int

func WebServer(ip string, port int) {

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:9000"},
	})

	goji.Use(c.Handler)

	goji.Get("/", func(c web.C, w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to the home page!")
	})

	goji.Get("/search", func(c web.C, w http.ResponseWriter, r *http.Request) {
		filter := db.Filter{}
		options := &db.Options{}

		r.ParseForm()
		startDate := r.Form.Get("startDate")
		endDate := r.Form.Get("endDate")
		limit := r.Form.Get("limit")

		if startDate != "" {
			filter.StartDate = startDate
		}

		if endDate != "" {
			filter.EndDate = endDate
		}

		if limit == "" {
			options.Limit = 50
		} else {
			limitUint64, err := strconv.Atoi(limit)
			if err != nil {
				options.Limit = 50
			} else {
				options.Limit = limitUint64
			}
		}

		order := r.Form["orderby"]
		if len(order) == 0 {
			options.Sort = append(options.Sort, "-datetime")
		} else {
			options.Sort = order
		}

		var results []db.DbObject

		db.Db.Find(&filter, options, &results)
		jsonResults, err := json.Marshal(results)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		fmt.Fprintf(w, "%s", string(jsonResults))
	})

	goji.Get("/call/:id", func(c web.C, w http.ResponseWriter, r *http.Request) {
		callId := c.URLParams["id"]
		fmt.Print(callId)
		filter := db.NewFilter()
		options := &db.Options{}

		filter.Equals["callid"] = callId
		options.Sort = append(options.Sort, "datetime")

		var results []db.DbObject
		db.Db.Find(&filter, options, &results)

		jsonResults, err := json.Marshal(results)
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		fmt.Fprintf(w, "%s", string(jsonResults))

	})

	goji.Serve()
}

func UDPServer(ip string, port int) {
	addr := net.UDPAddr{
		Port: port,
		IP:   net.ParseIP(ip),
	}
	fmt.Println("Flanders server listening on ", ip+":", port)
	conn, err := net.ListenUDP("udp", &addr)
	defer conn.Close()
	if err != nil {
		panic(err)
	}

	for {
		packet := make([]byte, 4096)

		length, _, err := conn.ReadFromUDP(packet)

		packet = packet[:length]
		// hepString := string(packet[:length])

		// fmt.Printf("\nPacket: %X\n", truncatedPacket)
		// fmt.Printf("\nPacket: %s\n", hepString)

		if err != nil {
			fmt.Println(err)
			continue
		}

		hepMsg, hepErr := hep.NewHepMsg(packet)

		if hepErr != nil {
			fmt.Println("ERROR PARSING HEP MESSAGE.................")
			fmt.Println(hepErr)
			continue
		}
		//fmt.Printf("%#v\n", hepMsg)

		datetime := time.Now()

		// Store HEP message in database
		dbObject := db.NewDbObject()
		dbObject.Datetime = datetime
		dbObject.MicroSeconds = datetime.Nanosecond() / 1000
		dbObject.Method = hepMsg.SipMsg.StartLine.Method + hepMsg.SipMsg.StartLine.Resp
		dbObject.ReplyReason = hepMsg.SipMsg.StartLine.RespText
		dbObject.SourceIp = hepMsg.Ip4SourceAddress
		dbObject.SourcePort = hepMsg.SourcePort
		dbObject.DestinationIp = hepMsg.Ip4DestinationAddress
		dbObject.DestinationPort = hepMsg.DestinationPort
		dbObject.CallId = hepMsg.SipMsg.CallId
		dbObject.FromUser = hepMsg.SipMsg.From.URI.User
		dbObject.FromDomain = hepMsg.SipMsg.From.URI.Host
		dbObject.FromTag = hepMsg.SipMsg.From.Tag
		dbObject.ToUser = hepMsg.SipMsg.To.URI.User
		dbObject.ToDomain = hepMsg.SipMsg.To.URI.Host
		dbObject.ToTag = hepMsg.SipMsg.To.Tag
		for _, header := range hepMsg.SipMsg.Headers {
			if header.Header == "x-cid" {
				dbObject.CallIdAleg = header.Val
			}
		}

		// dbObject.ContactUser = hepMsg.SipMsg.Contact.URI.User
		// dbOjbect.ContactIp =
		// dbOjbect.ContactPort =

		dbObject.Msg = hepMsg.SipMsg.Msg

		fmt.Printf("\n\nDbObject-----------\n%+v\n", dbObject)

		err = dbObject.Save()
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}
