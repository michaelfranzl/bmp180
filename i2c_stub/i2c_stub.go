// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package i2c_stub (part of project bmp180) provides an emulated BMP180 device
// that can be used to test functionality of the 'sensor' package when no
// physical BMP180 device is attached or available.

package i2c_stub

import "fmt"

type Devfs struct {
	Dev string
}

func Open(dev *Devfs, address byte) (*MockDevice, error) {
	return &MockDevice{}, nil
}

type MockDevice struct {
	regControl byte
}

func (d *MockDevice) Close() error {
	return nil
}

func (d *MockDevice) Read(buf []byte) error {
	return nil
}

func (d *MockDevice) ReadReg(reg byte, buf []byte) error {
	switch reg {
	case 0xD0:
		// This address holds the chip ID. BMP180 chips have ID 0x55.
		buf[0] = 0x55

	case 0xAA:
		// This address and the next 21 words hold the chip calibration data
		copy(buf, []byte{
			0x1e, 0xe7, // ac1 = 7911
			0xfc, 0x5a, // ac2 = -934
			0xc8, 0x1e, // ac3 = -14306
			0x7b, 0x4f, // ac4 = 31567
			0x64, 0x47, // ac5 = 25671
			0x4a, 0x1e, // ac6 = 18974
			0x15, 0x7a, // b1 = 5498
			0x00, 0x2e, // b2 = 46
			0x80, 0x00, // mb = -32768
			0xd4, 0xbd, // mc = -11075
			0x09, 0x80}) // md = 2432

	case 0xF6:
		// This address and the next 2 words hold the MSB, LSB, XLSB of either
		// raw temperature or pressure values, depending on what was measured.
		switch d.regControl {
		case 0x2E:
			// temp
			copy(buf, []byte{0x69, 0xEC})
		case 0x34, 0x74, 0xb4, 0xf4:
			// pressure
			copy(buf, []byte{0x98, 0x2F, 0xC0})
		default:
			panic(fmt.Sprintf("Reading from register 0xF6: Control register 0x%x", d.regControl))
		}

	default:
		panic(fmt.Sprintf("Reading from register 0x%x is not implemented", reg))
	}
	return nil
}

func (d *MockDevice) Write(buf []byte) (err error) {
	return nil
}

func (d *MockDevice) WriteReg(reg byte, buf []byte) (err error) {
	switch reg {
	case 0xF4:
		// Writing to this register either starts a temperature (0x2E)
		// or pressure (0x34 [oss=0], 0x74 [oss=1], 0xb4 [oss=2], 0xf4 [oss=3])
		// measurement.
		d.regControl = buf[0]
	}
	return nil
}
