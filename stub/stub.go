// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stub provides an emulated BMP180 I2C device which can be used
// to test functionality of the bmp180 package when no I2C bus or physical
// BMP180 device is available.
package stub

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"time"
)

const (
	regID             = 0xD0
	regControl        = 0xF4
	regTempOrPressure = 0xF6
	cmdReadTemp       = 0x2E
	cmdReadPressure   = 0x34
	regCalibration    = 0xAA
)

// Devfs is a stubbed I2C driver.
type Devfs struct {
	Dev string
}

// Open returns a pointer to BMP180StubDevice
func Open(dev *Devfs, address byte) (*BMP180StubDevice, error) {
	var sd BMP180StubDevice
	sd.numMeasurements = 0

	// set up the 256 word chip memory
	sd.memory = make([]byte, 0xff)

	// popuplate the 256 word chip memory

	// This address holds the chip ID. BMP180 chips have ID 0x55.
	sd.memory[regID] = 0x55

	// Address 0xAA and the next 21 words hold the chip calibration data
	// Calibration data taken from http://www.osengr.org/WxShield/Downloads/BMP085-Calcs.pdf
	copy(sd.memory[regCalibration:], []byte{
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

	return &sd, nil
}

// BMP180StubDevice represents a stubbed BMP180 I2C device.
type BMP180StubDevice struct {
	numMeasurements int64
	memory          []byte
	memPointer      int // current read/write location
}

// Close would close the device, but since this is a stub, it does nothing.
func (d *BMP180StubDevice) Close() error {
	return nil
}

// Read copies len(`buf`) bytes from internal memory starting from
// the last memory pointer into `buf`.
func (d *BMP180StubDevice) Read(buf []byte) error {
	d.ReadReg(byte(d.memPointer), buf)
	return nil
}

// ReadReg copies len(`buf`) bytes from internal memory starting from
// address `reg` into `buf`.
func (d *BMP180StubDevice) ReadReg(reg byte, buf []byte) error {
	d.memPointer = int(reg)
	d.memPointer += copy(buf, d.memory[d.memPointer:])
	//fmt.Printf("READ REG %x: %x\n", reg, buf)
	time.Sleep(10 * time.Millisecond) // simulate Linux kernel bus access time
	return nil
}

// Write copies `buf` into internal memory starting from the last memory pointer.
func (d *BMP180StubDevice) Write(buf []byte) (err error) {
	d.WriteReg(byte(d.memPointer), buf)
	return nil
}

// WriteReg copies `buf` into internal memory starting from address `reg`.
func (d *BMP180StubDevice) WriteReg(reg byte, buf []byte) (err error) {
	d.memPointer = int(reg)
	d.memPointer += copy(d.memory[d.memPointer:], buf)

	switch reg {
	case regControl: // address of control register
		// Writing to this register either starts a temperature (0x2E)
		// or pressure (0x34 [oss=0], 0x74 [oss=1], 0xb4 [oss=2], 0xf4 [oss=3])
		// measurement.
		d.numMeasurements++

		switch buf[0] {
		// simulate measurement
		case cmdReadTemp:
			// Read raw temperature
			var ret []byte
			if d.numMeasurements < 4 {
				// Return static temperature. Raw value taken from http://www.osengr.org/WxShield/Downloads/BMP085-Calcs.pdf
				// Make test (calling this only once or twice) get expected value.
				ret = []byte{0x69, 0xEC} // 27116
			} else {
				// Dynamic testing over longer periods to make the results more interesting.
				// Simulate temperature that changes like a sine wave over one full day
				secsElapsedToday := time.Now().Unix() % 86400
				dayProgress := float64(secsElapsedToday) / 86400.0 // 0..1
				cosine := math.Cos(dayProgress * 2 * math.Pi)      // one full sine wave over one full day, 0..1
				rawval := 27116 + (cosine * 2000)
				rawvalInt := int16(rawval) + int16(rand.Intn(10))
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, rawvalInt)
				ret = buf.Bytes()
			}
			copy(d.memory[regTempOrPressure:], ret)
		case 0x34, 0x74, 0xb4:
			// Return two bytes of pressure for oversampling settings 0, 1 and 2.
			// Raw value taken from http://www.osengr.org/WxShield/Downloads/BMP085-Calcs.pdf
			copy(d.memory[regTempOrPressure:], []byte{0x98, 0x2F})
		case 0xf4:
			// Return three bytes of pressure for oversampling setting 3.
			// Raw value taken from http://www.osengr.org/WxShield/Downloads/BMP085-Calcs.pdf
			copy(d.memory[regTempOrPressure:], []byte{0x98, 0x2F, 0xC0}) // 9973696
		}
		d.memory[regControl] &^= 0x20 // clear the SCO bit to indicate completed measurement
	}
	time.Sleep(10 * time.Millisecond) // simulate Linux kernel bus access time
	return nil
}
