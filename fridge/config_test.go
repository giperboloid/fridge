package fridge

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"github.com/giperboloid/fridgems/entities"
	"github.com/smartystreets/goconvey/convey"
	log "github.com/Sirupsen/logrus"
)

func TestGetTurned(t *testing.T) {

	convey.Convey("Should get valid value", t, func() {
		testConfig := NewConfiguration()
		testConfig.SetTurnedOn(false)
		convey.So(testConfig.GetTurnedOn(), convey.ShouldEqual, false)
	})
}

func TestSetTurned(t *testing.T) {
	convey.Convey("Should set valid value", t, func() {
		testConfig := NewConfiguration()
		testConfig.SetTurnedOn(false)
		convey.So(testConfig.GetTurnedOn(), convey.ShouldEqual, false)
	})
}

func TestGetCollectFreq(t *testing.T) {

	convey.Convey("Should get valid value", t, func() {
		testConfig := NewConfiguration()
		testConfig.SetCollectFreq(1000)
		convey.So(testConfig.GetCollectFreq(), convey.ShouldEqual, 1000)
	})
}

func TestSetCollectFreq(t *testing.T) {

	convey.Convey("Should set valid value", t, func() {
		testConfig := NewConfiguration()
		testConfig.SetCollectFreq(1000)
		convey.So(testConfig.GetCollectFreq(), convey.ShouldEqual, 1000)
	})
}


func TestGetSendFreq(t *testing.T) {
	testConfig := NewConfiguration()
	convey.Convey("Should get valid value", t, func() {
		testConfig.SetSendFreq(1000)
		convey.So(testConfig.GetSendFreq(), convey.ShouldEqual, 1000)
	})
}

func TestSetSendFreq(t *testing.T) {

	convey.Convey("Should set valid value", t, func() {
		testConfig := NewConfiguration()
		testConfig.SetSendFreq(1000)
		convey.So(testConfig.GetSendFreq(), convey.ShouldEqual, 1000)
	})
}

func TestAddSubIntoPool(t *testing.T) {
	ch := make(chan struct{})
	key := "19-29"

	convey.Convey("Subscribe should add chan into the pool", t, func() {
		testConfig := NewConfiguration()
		testConfig.Subscribe(key, ch)
		convey.So(testConfig.SubsPool[key], convey.ShouldEqual, ch)
	})
}

func TestRemoveSubFromPool(t *testing.T) {
	ch := make(chan struct{})
	key := "19-29"

	convey.Convey("Unsubscribe should remove chan from the pool", t, func() {
		testConfig := NewConfiguration()
		testConfig.Subscribe(key, ch)

		testConfig.Unsubscribe(key)
		convey.So(testConfig.SubsPool[key], convey.ShouldEqual, nil)
	})
}

func TestUpdateConfig(t *testing.T) {
	maskOsArgs()

	exCfg := entities.FridgeConfig{
		TurnedOn:    true,
		SendFreq:    100,
		CollectFreq: 50}

	convey.Convey("UpdateConfig should updateConfig struct by new struct's values", t, func() {
		testConfig := NewConfiguration()
		testConfig.updateConfig(exCfg)
		convey.So(testConfig.GetTurnedOn(), convey.ShouldEqual, exCfg.TurnedOn)
		convey.So(testConfig.GetCollectFreq(), convey.ShouldEqual, exCfg.CollectFreq)
		convey.So(testConfig.GetSendFreq(), convey.ShouldEqual, exCfg.SendFreq)
	})
}


func TestListenConfig(t *testing.T) {
	maskOsArgs()

	cfg := entities.FridgeConfig{
		TurnedOn:    true,
		CollectFreq: 1000,
		SendFreq:    5000}

	connTypeConf := "tcp"
	hostConf := "localhost"
	portConf := "3000"

	convey.Convey("ListenConfig should receive a configuration", t, func() {

		ln, _ := net.Listen(connTypeConf, hostConf+":"+portConf)
		go func() {
			defer ln.Close()
			server, err := ln.Accept()
			if err != nil {
				t.Fail()
			}
			err = json.NewEncoder(server).Encode(cfg)
			if err != nil {
				t.Fail()
			}
		}()

		client, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fail()
		}
		testConfig := NewConfiguration()

		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}
		}()
		listenConfig(testConfig, client)

		convey.So(testConfig.GetSendFreq(), convey.ShouldEqual, 5000)
		convey.So(testConfig.GetCollectFreq(), convey.ShouldEqual, 1000)
		convey.So(testConfig.GetTurnedOn(), convey.ShouldEqual, true)
	})
}

func TestInit(t *testing.T) {
	maskOsArgs()
	devCfg := entities.FridgeConfig{
		TurnedOn:    true,
		CollectFreq: 1000,
		SendFreq:    5000}

	connTypeConf := "tcp"
	hostConf := "localhost"
	portConf := "3000"

	convey.Convey("Init should receive config", t, func() {
		control := &entities.RoutinesController{StopChan: make(chan struct{})}
		ln, _ := net.Listen(connTypeConf, hostConf+":"+portConf)
		go func() {
			defer ln.Close()
			server, err := ln.Accept()
			if err != nil {
				t.Fail()
			}
			err = json.NewEncoder(server).Encode(devCfg)
			if err != nil {
				t.Fail()
			}
		}()
		testConfig := NewConfiguration()

		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}} ()
		testConfig.RequestConfig(connTypeConf, hostConf, portConf, control, maskOsArgs())

		convey.So(testConfig.GetSendFreq(), convey.ShouldEqual, 5000)
		convey.So(testConfig.GetCollectFreq(), convey.ShouldEqual, 1000)
		convey.So(testConfig.GetTurnedOn(), convey.ShouldEqual, true)
	})
}

func maskOsArgs() []string {
	os.Args = []string{"cmd", "fridgeconfig", "LG", "00-00-00-00-00-00"}
	return os.Args
}
