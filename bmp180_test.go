// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmp180_test

import (
	"fmt"

	"github.com/michaelfranzl/bmp180"
	"github.com/michaelfranzl/bmp180/stub"
	"golang.org/x/exp/io/i2c"
)

// This example shows the basic usage of this package
func Example() {
	var (
		err       error
		i2cDevice bmp180.Device
	)

	// Specify the path to a I2C device node provided by the Linux kernel, e.g. "/dev/i2c-1"
	// This example uses "/dev/null" so that the i2c_stub device is guaranteed to be used
	// and this test is guarenteed to pass.
	devfs := i2c.Devfs{Dev: "/dev/null"}

	// Open an I2C device that can communicate with the sensor at bus address 0x77
	i2cDevice, err = i2c.Open(&devfs, 0x77)

	if err != nil {
		// If no physical sensor available, use a stubbed I2C Device which is
		// provided in this package.
		devfsStub := stub.Devfs{Dev: "/dev/i2c-1"}
		i2cDevice, err = stub.Open(&devfsStub, 0x77)
	}

	defer func() {
		fmt.Println("Closing")
		i2cDevice.Close()
	}()

	myBMP180 := bmp180.NewSensor(i2cDevice)

	id, _ := myBMP180.ID()
	t, _ := myBMP180.Temperature()
	p, _ := myBMP180.Pressure(3)
	p0, _ := myBMP180.PressureSealevel(3, 500)

	fmt.Printf("ID=0x%x t=%.3f°C p=%.3fmbar p0=%.3fmbar\n", id, t, p, p0)

	// Output:
	// ID=0x55 t=23.776°C p=980.046mbar p0=1040.241mbar
	// Closing
}
