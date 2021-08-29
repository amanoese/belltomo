// This is a sensor station that uses a ESP8266 or ESP32 running on the device UART1.
// It creates an MQTT connection that publishes a message every second
// to an MQTT broker.
//
// In other words:
// Your computer <--> UART0 <--> MCU <--> UART1 <--> ESP8266 <--> Internet <--> MQTT broker.
//
// You must install the Paho MQTT package to build this program:
//
// 		go get -u github.com/eclipse/paho.mqtt.golang
//
package main

import (
	"fmt"
	"github.com/amanoese/belltomo/config"
	"machine"
	"math/rand"
	"time"
	"tinygo.org/x/drivers/hd44780i2c"
	"tinygo.org/x/drivers/net/mqtt"
	"tinygo.org/x/drivers/wifinina"
)

// access point info
var ssid = config.SSID
var pass = config.PASS

// IP address of the MQTT broker to use. Replace with your own info.
const server = "tcp://test.mosquitto.org:1883"

//const server = "ssl://test.mosquitto.org:8883"

// change these to connect to a different UART or pins for the ESP8266/ESP32
var (
	// these are the default pins for the Arduino Nano33 IoT.
	spi = machine.NINA_SPI

	// this is the ESP chip that has the WIFININA firmware flashed on it
	adaptor *wifinina.Device

	cl      mqtt.Client
	topicTx = "tinygo/tx"
	topicRx = "tinygo/rx"
)

func lcdDisp(lcd *hd44780i2c.Device, msg string) {
	lcd.ClearDisplay()
	time.Sleep(20 * time.Millisecond)

	if msg == "unko" {
		lcd.CreateCharacter(0x0, []byte{0x01, 0x03, 0x04, 0x07, 0x08, 0x0F, 0x10, 0x1F})
		lcd.CreateCharacter(0x1, []byte{0x10, 0x18, 0x04, 0x1C, 0x02, 0x1E, 0x01, 0x1F})
		lcd.Print([]byte("    "))
		lcd.Print([]byte{0x0, 0x1})
		lcd.Print([]byte(msg))
		lcd.Print([]byte{0x0, 0x1})
		return
	}

	lcd.Print([]byte(msg))
}

func mLcdDisp(lcd *hd44780i2c.Device) func(msg string) {
	return func(msg string) {
		lcdDisp(lcd, msg)
	}
}

func getSubHandler(lcd *hd44780i2c.Device) func(client mqtt.Client, msg mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		topic := msg.Topic()
		payload := msg.Payload()
		str := fmt.Sprintf("%s", payload)

		fmt.Printf("[%s]  ", topic)
		fmt.Printf("%s\r\n", payload)

		lcdDisp(lcd, str)
	}
}

func main() {
	machine.I2C0.Configure(machine.I2CConfig{
		Frequency: machine.TWI_FREQ_400KHZ,
	})
	lcd := hd44780i2c.New(machine.I2C0, 0x3F) // some modules have address 0x3F

	lcd.Configure(hd44780i2c.Config{
		Width:       16, // required
		Height:      2,  // required
		CursorOn:    false,
		CursorBlink: false,
	})

	time.Sleep(3000 * time.Millisecond)

	rand.Seed(time.Now().UnixNano())

	// Configure SPI for 8Mhz, Mode 0, MSB First
	spi.Configure(machine.SPIConfig{
		Frequency: 8 * 1e6,
		SDO:       machine.NINA_SDO,
		SDI:       machine.NINA_SDI,
		SCK:       machine.NINA_SCK,
	})

	// Init esp8266/esp32
	adaptor = wifinina.New(spi,
		machine.NINA_CS,
		machine.NINA_ACK,
		machine.NINA_GPIO0,
		machine.NINA_RESETN)
	adaptor.Configure()

	display := mLcdDisp(&lcd)
	display("connect to AP...")
	connectToAP()
	display("connected AP")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server).SetClientID("tinygo-client-" + randomString(10))

	println("Connecting to MQTT broker at", server)
	display("Connect MQTT broker...")
	cl = mqtt.NewClient(opts)
	if token := cl.Connect(); token.Wait() && token.Error() != nil {
		failMessage(token.Error().Error())
	}

	subHander := getSubHandler(&lcd)
	// subscribe
	token := cl.Subscribe(topicRx, 0, subHander)
	token.Wait()
	if token.Error() != nil {
		failMessage(token.Error().Error())
	}

	display("Subscribe...")
	go loop()

	select {}

	// Right now this code is never reached. Need a way to trigger it...
	println("Disconnecting MQTT...")
	cl.Disconnect(100)

	println("Done.")
}

func loop() {
	for i := 0; ; i++ {
		//println("...")
		//display("Subscribe...")
		time.Sleep(3000 * time.Millisecond)
	}
}

// connect to access point
func connectToAP() {
	time.Sleep(2 * time.Second)
	println("Connecting to " + ssid)
	adaptor.SetPassphrase(ssid, pass)
	for st, _ := adaptor.GetConnectionStatus(); st != wifinina.StatusConnected; {
		println("Connection status: " + st.String())
		time.Sleep(1 * time.Second)
		st, _ = adaptor.GetConnectionStatus()
	}
	println("Connected.")
	time.Sleep(2 * time.Second)
	ip, _, _, err := adaptor.GetIP()
	for ; err != nil; ip, _, _, err = adaptor.GetIP() {
		println(err.Error())
		time.Sleep(1 * time.Second)
	}
	println(ip.String())
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

// Generate a random string of A-Z chars with len = l
func randomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(randomInt(65, 90))
	}
	return string(bytes)
}

func failMessage(msg string) {
	for {
		println(msg)
		time.Sleep(1 * time.Second)
	}
}
