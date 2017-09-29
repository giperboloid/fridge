package fridge

import (
	"os"
	"time"
	"reflect"
	"github.com/giperboloid/devicems/models"
	"testing"
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
		Action: "update",
		Meta: entities.Metadata{
			Type: os.Args[1],
			Name: os.Args[2],
			MAC:  os.Args[3]},
		Data: entities.FridgeData{
			TempCam1: topMap,
			TempCam2: botMap},
	}

	convey.Convey("DataGenerator should produce structs with data", t, func() {

		go DataCollector(ticker, bot, top, reqChan, stopInner)
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
		Action: "update",
		Meta: entities.Metadata{
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

func maskOsArgs() {
	os.Args = []string{"cmd", "fridgeconfig", "LG", "00-00-00-00-00-00"}
}

