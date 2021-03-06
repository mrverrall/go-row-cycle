package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/mrverrall/go-row/peripheral"
	"github.com/mrverrall/go-row/pm5"
)

var (
	deviceName = "go-row"
)

func main() {

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Printf("signal received from os: %s", sig)
		unsetBT()
		done <- true
	}()
	go btWorker(done)
	<-done
}

func btWorker(done chan bool) {
	for {
		unsetBT()
		d, err := linux.NewDeviceWithName(deviceName)
		if err != nil {
			log.Printf("can't get  BT device: %s", err)
			done <- true
		}
		ble.SetDefaultDevice(d)

		log.Printf("searching for PM5...")
		rower, err := pm5.NewClient()

		if err != nil {
			log.Printf("PM5 error: %s", err)
			continue
		}

		sensors := peripheral.Sensors{
			peripheral.NewCyclePower(deviceName),
			peripheral.NewRunningSpeed(deviceName),
			peripheral.NewHRM(deviceName),
		}

		log.Println("advertising sensor services")
		go ble.AdvertiseNameAndServices(context.Background(), deviceName, sensors.UUIDs()...)

		for data := range rower.StatusCh {
			for _, s := range sensors {
				select {
				case s.DataCh <- data:
				default:
				}
			}
		}
	}
}

func unsetBT() {
	ble.Stop()
	time.Sleep(time.Second * 5)
}
