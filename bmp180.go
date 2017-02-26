// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bmp180 provides reading environmental data from the BMP180
// family of environmental (temperature, pressure) I2C sensors.
package bmp180

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// Device is the interface that groups methods used to communicate with an
// underlying I2C device of any kind.
type Device interface {
	Close() error
	Read(buf []byte) error
	ReadReg(reg byte, buf []byte) error
	Write(buf []byte) (err error)
	WriteReg(reg byte, buf []byte) (err error)
}

const (
	regID               = 0xD0
	regControl          = 0xF4
	regTempOrPressure   = 0xF6
	cmdReadTemp         = 0x2E
	cmdReadPressure     = 0x34
	regCalibration      = 0xAA
	calibrationNumWords = 22
)

type calibration struct {
	ac1, ac2, ac3      int16
	ac4, ac5, ac6      uint16
	b1, b2, mb, mc, md int16
}

type constants struct {
	c5, c6     float64
	mc, md     float64
	x0, x1, x2 float64
	y0, y1, y2 float64
	p0, p1, p2 float64
}

// Sensor is a struct which holds methods and data to communmicate with a
// BMP180 environmental sensor.
type Sensor struct {
	dev         Device      // the I2C communication device
	calib       calibration // parsed calibration data
	cnsts       constants   // unchanging constants calculated from calibration data
	tempCelsius float64     // the latest measured temperature
}

// NewSensor returns a pointer to a variable of type Sensor which has the I2C
// communication device set. It reads calibration data from the I2C device
// and pre-calculates constants needed for temperature and pressure calulations.
func NewSensor(device Device) *Sensor {
	s := new(Sensor)
	s.dev = device

	readCalibration(s)
	s.cnsts = calcConstants(s.calib)
	return s
}

// ID reads the chip ID from the underlying I2C device. The BMP180 sensor
// always returns 0x55. This function can be used to test basic communication.
func (s *Sensor) ID() (byte, error) {
	buf := make([]byte, 1, 1)
	err := s.dev.ReadReg(regID, buf)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// Temperature reads a raw temperature value from the underlying IC2 device.
// It then calculates a real temperature in degrees Celsius based on calibration
// data, stores the temperature as state information for the next pressure calculation
// and returns the temperature.
func (s *Sensor) Temperature() (float64, error) {
	tu, err := readRawTemp(s)
	if err != nil {
		return 0, err
	}
	t := calcTempCelsius(tu, s.cnsts)
	s.tempCelsius = t
	return t, nil
}

// Pressure reads a raw pressure value from the underlying IC2 device.
// From this it then calculates a real pressure in millibars based on the
// previously read temperature and calibration data and returns that pressure.
func (s *Sensor) Pressure(oss uint8) (float64, error) {
	msb, lsb, xlsb, err := readRawPressure(s, oss)
	if err != nil {
		return 0, err
	}
	return calcPressurePascal(s.tempCelsius, msb, lsb, xlsb, s.cnsts), nil
}

// PressureSealevel does the same as Pressure, but calculates and returns the
// sealevel pressure based on altitudeMeters at which the pressure has been measured.
func (s *Sensor) PressureSealevel(oss uint8, altitudeMeters float64) (float64, error) {
	p, err := s.Pressure(oss)
	if err != nil {
		return 0, err
	}
	return calcPressurePascalSealevel(p, altitudeMeters), nil
}

func readRawTemp(s *Sensor) (uint16, error) {
	err := s.dev.WriteReg(regControl, []byte{cmdReadTemp})

	time.Sleep(5 * time.Millisecond)

	buf := make([]byte, 2, 2)
	err = s.dev.ReadReg(regTempOrPressure, buf)
	if err != nil {
		return 0, err
	}
	return uint16(buf[0])<<8 + uint16(buf[1]), nil
}

func readRawPressure(s *Sensor, oss uint8) (msb byte, lsb byte, xlsb byte, err error) {

	var cmd byte
	cmd = cmdReadPressure + (oss << 6)

	t1 := time.Now()
	err = s.dev.WriteReg(regControl, []byte{cmd}) // start the measurement
	if err != nil {
		return
	}
	accessDuration := time.Since(t1)

	// Testing a real sensor with I2C bus clock of 100kHz on a Raspberry PI 2 shows that
	// ReadReg blocks about 9ms (Linux bus access) + 2ms per transmitted word.
	// Thus, we minimize calls to ReadReg and read as many bytes as possible at once.

	// fmt.Printf("WRITE TIME %v\n", accessDuration)

	buf := make([]byte, 5) // storage for registers 0xF4..0xF8

	var delay time.Duration
	// typical measurement times from Bosch BMP180 datasheet
	switch oss {
	case 0:
		delay = 4500
	case 1:
		delay = 7500
	case 2:
		delay = 13500
	case 3:
		delay = 25500
	}

	// We sleep for the suggested measurement time minus the bus access time which acts as an 'implicit sleep'
	time.Sleep(delay*time.Microsecond - accessDuration)

	//t2 := time.Now() // debug

	err = s.dev.ReadReg(regControl, buf) // read registers 0xF4..0xF8 (9ms + 5 * 2ms = 19ms)
	if err != nil {
		return
	}

	// The chip will change the control register 0xF4 when measurement is done. We poll for this.
	// Because we slept for the suggested time, this loop should not run, but it is here
	// in case there is an unexpected delay in the conversion.
	for buf[0]&0x20 > 0 {
		// wait for SCO bit to be cleared
		//fmt.Printf("LOOP %v %b\n", time.Since(t1), buf[0])
		err = s.dev.ReadReg(regControl, buf) // read registers 0xF4..0xF8 (9ms + 5 * 2ms = 19ms)
		if err != nil {
			return
		}
	}

	// At this point, measurement is guaranteed to be completed
	//fmt.Printf("DONE %v\n", time.Since(t2)) // debug

	msb = buf[2]  // register 0xF6
	lsb = buf[3]  // register 0xF7
	xlsb = buf[4] // register 0xF8
	return
}

func readCalibration(s *Sensor) (err error) {
	buf := make([]byte, calibrationNumWords, calibrationNumWords)
	err = s.dev.ReadReg(regCalibration, buf)
	if err != nil {
		return
	}

	s.calib.ac1 = readInt16(buf[0:2])
	s.calib.ac2 = readInt16(buf[2:4])
	s.calib.ac3 = readInt16(buf[4:6])

	s.calib.ac4 = readUint16(buf[6:8])
	s.calib.ac5 = readUint16(buf[8:10])
	s.calib.ac6 = readUint16(buf[10:12])

	s.calib.b1 = readInt16(buf[12:14])
	s.calib.b2 = readInt16(buf[14:16])
	s.calib.mb = readInt16(buf[16:18])
	s.calib.mc = readInt16(buf[18:20])
	s.calib.md = readInt16(buf[20:22])

	return nil
}

func readInt16(data []byte) (ret int16) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &ret)
	return
}

func readUint16(data []byte) (ret uint16) {
	buf := bytes.NewBuffer(data)
	binary.Read(buf, binary.BigEndian, &ret)
	return
}
