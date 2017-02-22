// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bmp180 provides an interface to the BMP180 family of environmental
// (temperature, pressure) sensors attached to an I2C interface.
package bmp180

import (
	"bytes"
	"encoding/binary"
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
	err := s.dev.ReadReg(0xD0, buf)
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
	err = s.dev.WriteReg(regControl, []byte{cmd})
	if err != nil {
		return
	}

	var delay time.Duration
	switch oss {
	case 0:
		delay = 5
	case 1:
		delay = 8
	case 2:
		delay = 14
	case 3:
		delay = 26
	}

	time.Sleep(delay * time.Millisecond)

	buf := make([]byte, 3, 3)
	err = s.dev.ReadReg(regTempOrPressure, buf)
	if err != nil {
		return
	}
	msb = buf[0]
	lsb = buf[1]
	xlsb = buf[2]
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
