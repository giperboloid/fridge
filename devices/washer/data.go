package washer

import (
	"time"
	"github.com/giperboloid/devicems/models"
	"github.com/giperboloid/devicems/config/washerconfig"
	"os"
	"net"
	"encoding/json"
	"github.com/KharkivGophers/device-smart-house/tcp/connectionupdate"
)

//DataCollector gathers data from DataGenerator
//and sends completed request's structures to the ReqChan channel
func DataCollector(ticker *time.Ticker, turnOversStorage <-chan entities.GenerateWasherData, waterTempStorage <-chan entities.GenerateWasherData,
	RequestStorage chan entities.WasherRequest) {

	var requestturnOversStorage = make(map[int64]int64)
	var requestwaterTempStorage = make(map[int64]float32)

	for {
		select {
		case tv := <-waterTempStorage:
			requestwaterTempStorage[tv.Time] = tv.WaterTemp
		case bv := <-turnOversStorage:
			requestturnOversStorage[bv.Time] = bv.Turnovers
		case <-ticker.C:
			RequestStorage <-constructReq(requestturnOversStorage, requestwaterTempStorage)

			requestwaterTempStorage = make(map[int64]float32)
			requestturnOversStorage = make(map[int64]int64)
		}
	}
}

//RunDataCollector setups DataCollector
func RunDataCollector(config *washerconfig.WasherConfig, turnOversStorage <-chan entities.GenerateWasherData,
	waterTempStorage <-chan entities.GenerateWasherData, RequestStorage chan entities.WasherRequest) {
	washTime := config.GetWashTime()
	rinseTime := config.GetRinseTime()
	spinTime := config.GetSpinTime()
	ticker := time.NewTicker(time.Second * 5)

	timer := time.NewTimer(time.Second * time.Duration(washTime + rinseTime + spinTime))
	go DataCollector(ticker, turnOversStorage, waterTempStorage, RequestStorage)
	<-timer.C
	ticker.Stop()
}

func constructReq(turnOversStorage map[int64]int64, waterTempStorage map[int64]float32) entities.WasherRequest {

	var washerData entities.WasherData
	args := os.Args[1:]

	washerData.WaterTemp = waterTempStorage
	washerData.Turnovers = turnOversStorage

	request := entities.WasherRequest{
		Action: "update",
		Time: time.Now().UnixNano(),
		Meta: entities.Metadata{
			Type: args[0],
			Name: args[1],
			MAC:  args[2]},
		Data: washerData,
	}

	return request
}

// DataGenerator generates pseudo-random numbers
func DataGenerator(stage string, ticker *time.Ticker, maxTemperature int64, maxTurnovers int64, turnOversStorage chan<- entities.GenerateWasherData,
	waterTempStorage chan<- entities.GenerateWasherData) {

	log.Info(stage, " -------- S T A R T E D!")
	for {
		select {
		case <-ticker.C:
			turnOversStorage <- entities.GenerateWasherData{Time: makeTimestamp(), Turnovers: int64(rand.Intn(int(maxTurnovers)))}
			waterTempStorage <- entities.GenerateWasherData{Time: makeTimestamp(), WaterTemp: rand.Float32() * float32(maxTemperature)}
		}
	}
}

func RunDataGenerator(config *washerconfig.WasherConfig, turnOversStorage chan<- entities.GenerateWasherData,
	waterTempStorage chan<- entities.GenerateWasherData, c *entities.Control, firstStep chan struct{}) {

	maxTemperature := config.GetTemperature()

	// Run wash
	washTime := config.GetWashTime()
	maxWashTurnovers := config.GetWashTurnovers()
	stageWash := "W A S H"
	ticker := time.NewTicker(time.Second * 3)
	timer := time.NewTimer(time.Second * time.Duration(washTime))
	go DataGenerator(stageWash, ticker, int64(maxTemperature), maxWashTurnovers, turnOversStorage, waterTempStorage)
	<-timer.C
	ticker.Stop()
	log.Info(stageWash, " -------- F I N I S H E D!")

	// Run rinse
	rinseTime := config.GetRinseTime()
	maxRinseTurnovers := config.GetRinseTurnovers()
	stageRinse := "R I N S E"
	ticker = time.NewTicker(time.Second * 3)
	timer = time.NewTimer(time.Second * time.Duration(rinseTime))
	go DataGenerator(stageRinse, ticker, int64(maxTemperature), maxRinseTurnovers, turnOversStorage, waterTempStorage)
	<-timer.C
	ticker.Stop()
	log.Info(stageRinse, " -------- F I N I S H E D!")

	// Run spin
	spinTime := config.GetSpinTime()
	maxSpinTurnovers := config.GetSpinTurnovers()
	stageSpin := "S P I N"
	ticker = time.NewTicker(time.Second * 3)
	timer = time.NewTimer(time.Second * time.Duration(spinTime))
	go DataGenerator(stageSpin, ticker, int64(maxTemperature), maxSpinTurnovers, turnOversStorage, waterTempStorage)
	<-timer.C
	ticker.Stop()
	log.Info(stageSpin, " -------- F I N I S H E D!")
	log.Warn("W A S H I N G --- M A C H I N E --- F I N I S H E D")

	firstStep <- struct{}{}
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// Connection
func GetDial(connType string, host string, port string) net.Conn {
	var times int
	conn, err := net.Dial(connType, host+":"+port)

	for err != nil {
		if times >= 5 {
			panic("Can't connect to the server: send")
		}
		time.Sleep(time.Second)
		conn, err = net.Dial(connType, host+":"+port)
		error.CheckError("getDial()", err)
		times++
		log.Warningln("Reconnect times: ", times)
	}
	return conn
}

func Send(r entities.WasherRequest, conn net.Conn) {
	var resp entities.Response
	r.Time = time.Now().UnixNano()

	err := json.NewEncoder(conn).Encode(r)

	if err != nil {
		panic("Nothing to encode")
	}
	error.CheckError("send(): JSON Encode: ", err)

	err = json.NewDecoder(conn).Decode(&resp)
	if err != nil {
		panic("No response found")
	}
	error.CheckError("send(): JSON Decode: ", err)

	log.Infoln("Data was sent; Response from center: ", resp)
}

//DataTransfer func sends request as JSON to the centre
func DataTransfer(config *washerconfig.WasherConfig, requestStorage chan entities.WasherRequest, c *entities.Control) {

	// for data transfer
	transferConnParams := entities.TransferConnParams{
		HostOut: connectionupdate.GetEnvCenter("CENTER_TCP_ADDR"),
		PortOut: "3030",
		ConnTypeOut: "tcp",
	}

	defer func() {
		if a := recover(); a != nil {
			log.Error(a)
			c.Close()
		}
	} ()
	conn := GetDial(transferConnParams.ConnTypeOut, transferConnParams.HostOut, transferConnParams.PortOut)

	for {
		select {
		case r := <-requestStorage:
			go func() {
				defer func() {
					if a := recover(); a != nil {
						log.Error(a)
						c.Close()
					}
				} ()
				Send(r, conn)
			}()
		case <- c.Controller:
			log.Error("Data Transfer Failed")
			return
		}
	}
}
