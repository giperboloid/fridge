package fridge

import (
	"encoding/json"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/giperboloid/fridgems/entities"
	"github.com/smartystreets/goconvey/convey"
)

func TestDataCollector(t *testing.T) {
	maskOsArgs()
	var req entities.FridgeRequest
	ticker := time.NewTicker(time.Millisecond)
	top := make(chan entities.FridgeGenerData)
	bot := make(chan entities.FridgeGenerData)
	reqChan := make(chan entities.FridgeRequest)
	stopInner := make(chan struct{})

	botMap := make(map[int64]float32)
	topMap := make(map[int64]float32)

	topMap[0] = 1.01

	botMap[0] = 10.01

	exReq := entities.FridgeRequest{
		Action: "updateConfig",
		Meta: entities.DevMeta{
			Type: os.Args[1],
			Name: os.Args[2],
			MAC:  os.Args[3]},
		Data: entities.FridgeData{
			TempCam1: topMap,
			TempCam2: botMap},
	}

	convey.Convey("dataGenerator should produce structs with data", t, func() {

		go dataCollector(ticker, bot, top, reqChan, stopInner)
		top <- entities.FridgeGenerData{Data: 1.01}
		bot <- entities.FridgeGenerData{Data: 10.01}

		time.Sleep(time.Millisecond * 10)

		req = <-reqChan

		//Compare struct's data
		b := reflect.DeepEqual(req.Data, exReq.Data)
		convey.So(b, convey.ShouldEqual, true)
	})
}

func TestConstructReq(t *testing.T) {
	os.Args = []string{"cmd", "fridgeconfig", "LG", "00-00-00-00-00-00"}
	var exReq entities.FridgeRequest
	bot := make(map[int64]float32)
	top := make(map[int64]float32)

	bot[1] = 1.01
	bot[2] = 2.02
	bot[3] = 3.03

	top[1] = 10.01
	top[2] = 20.01
	top[3] = 30.01

	exReq = entities.FridgeRequest{
		Action: "updateConfig",
		Meta: entities.DevMeta{
			Type: os.Args[1],
			Name: os.Args[2],
			MAC:  os.Args[3]},
		Data: entities.FridgeData{TempCam1: top, TempCam2: bot},
	}
	convey.Convey("ConstructReq should produce Request struct with received data", t, func() {
		req := constructReq(top, bot)
		b := reflect.DeepEqual(req.Data, exReq.Data)
		convey.So(req.Action, convey.ShouldEqual, exReq.Action)
		//Compare struct
		convey.So(b, convey.ShouldEqual, true)
		convey.So(req.Meta.MAC, convey.ShouldEqual, exReq.Meta.MAC)
		convey.So(req.Meta.Name, convey.ShouldEqual, exReq.Meta.Name)
		convey.So(req.Meta.Type, convey.ShouldEqual, exReq.Meta.Type)
	})
}

func TestDataGenerator(t *testing.T) {
	ticker := time.NewTicker(time.Millisecond)
	top := make(chan entities.FridgeGenerData)
	bot := make(chan entities.FridgeGenerData)
	stopInner := make(chan struct{})

	convey.Convey("dataGenerator should produce structs with data", t, func() {
		var fromTop, fromBot entities.FridgeGenerData
		var okTop, okBot bool

		go dataGenerator(ticker, bot, top, stopInner)
		fromTop, okTop = <-top
		fromBot, okBot = <-bot

		time.Sleep(time.Millisecond * 10)

		convey.So(okTop, convey.ShouldEqual, true)
		convey.So(okBot, convey.ShouldEqual, true)
		convey.So(fromBot.Data, convey.ShouldNotEqual, 0)
		convey.So(fromTop.Data, convey.ShouldNotEqual, 0)
		convey.So(reflect.TypeOf(fromBot.Data).String(), convey.ShouldEqual, "float32")
		convey.So(reflect.TypeOf(fromTop.Data).String(), convey.ShouldEqual, "float32")
	})
}

func TestMakeTimeStamp(t *testing.T) {
	convey.Convey("MakeTimeStamp should return timestamp as int64", t, func() {
		t := makeTimestamp()
		convey.So(reflect.TypeOf(t).String(), convey.ShouldEqual, "int64")
		convey.So(t, convey.ShouldNotBeEmpty)
		convey.So(t, convey.ShouldNotEqual, 0)
	})
}

//how to change conn configs?
func TestDataTransfer(t *testing.T) {
	maskOsArgs()
	connTypeOut := "tcp"
	hostOut := "localhost"
	portOut := "3030"

	bot := make(map[int64]float32)
	top := make(map[int64]float32)

	bot[1] = 1.01
	bot[2] = 2.02
	bot[3] = 3.03

	top[1] = 10.01
	top[2] = 20.01
	top[3] = 30.01

	var req entities.FridgeRequest
	exReq := entities.FridgeRequest{
		Action: "updateConfig",
		Meta: entities.DevMeta{
			Type: os.Args[1],
			Name: os.Args[2],
			MAC:  os.Args[3]},
		Data: entities.FridgeData{
			TempCam1: top,
			TempCam2: bot},
	}

	ch := make(chan entities.FridgeRequest)

	convey.Convey("DataSender should receive req from chan and transfer it to the server", t, func() {
		ln, err := net.Listen(connTypeOut, hostOut+":"+portOut)
		if err != nil {
			//t.Fail()
			panic("DataSender() Listen: error")
		}

		control := &entities.RoutinesController{StopChan:make(chan struct{})}
		go func() {
			defer ln.Close()
			server, err := ln.Accept()
			if err != nil {
				t.Fail()
			}
			err = json.NewDecoder(server).Decode(&req)
			if err != nil {
				t.Fail()
			}
		}()

		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}
		}()
		go DataSender(entities.Server{Host: hostOut}, ch, control)

		ch <- exReq

		time.Sleep(time.Millisecond * 10)
		b := reflect.DeepEqual(req.Data, exReq.Data)
		convey.So(req.Action, convey.ShouldEqual, exReq.Action)
		//Compare struct
		convey.So(b, convey.ShouldEqual, true)
		convey.So(req.Meta.MAC, convey.ShouldEqual, exReq.Meta.MAC)
		convey.So(req.Meta.Name, convey.ShouldEqual, exReq.Meta.Name)
		convey.So(req.Meta.Type, convey.ShouldEqual, exReq.Meta.Type)
	})
}

func TestGetDial(t *testing.T) {
	connTypeConf := "tcp"
	hostConf := "0.0.0.0"
	portConf := "3000"

	convey.Convey("tcp tcp should be established", t, func() {
		ln, _ := net.Listen(connTypeConf, hostConf+":"+portConf)
		conn := Dial(connTypeConf, hostConf, portConf)
		time.Sleep(time.Millisecond * 100)
		defer ln.Close()
		defer conn.Close()
		convey.So(conn, convey.ShouldNotBeNil)
	})
}

func TestSend(t *testing.T) {
	os.Args = []string{"cmd", "fridgeconfig", "LG", "00-00-00-00-00-00"}
	var req entities.FridgeRequest
	var resp entities.Response

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	exReq := entities.FridgeRequest{
		Action: "updateConfig",
		Meta: entities.DevMeta{
			Type: os.Args[1],
			Name: os.Args[2],
			MAC:  os.Args[3]},
	}

	resp = entities.Response{Descr: "Struct has been received"}
	convey.Convey("Send should send JSON to the server", t, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}
		} ()
		go Send(exReq, client) // request counter is missing

		json.NewDecoder(server).Decode(&req)
		json.NewEncoder(server).Encode(resp)

		convey.So(req.Action, convey.ShouldEqual, exReq.Action)
		convey.So(req.Meta.MAC, convey.ShouldEqual, exReq.Meta.MAC)
		convey.So(req.Meta.Name, convey.ShouldEqual, exReq.Meta.Name)
		convey.So(req.Meta.Type, convey.ShouldEqual, exReq.Meta.Type)
	})
}