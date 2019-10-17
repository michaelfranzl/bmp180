// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The following calculations are based on a paper hosted here:
// http://www.osengr.org/WxShield/Downloads/BMP085-Calcs.pdf

package bmp180

import (
	"math"
)

func calcTempCelsius(ut uint16, calib calibration) float64 {
	x1 := int(ut - calib.ac6) * int(calib.ac5) / (1 << 15)
	x2 := int(calib.mc) * (1 << 11) / (x1 + int(calib.md))
	b5 := x1 + x2
	t := float64((b5 + 8) / (1 << 4)) / 10.0
	return t
}

func calcPressurePascal(ut uint16, tempCelsius float64, msb, lsb, xlsb byte, oss uint8, calib calibration) float64 {
	msbInt := int64(msb)
	lsbInt := int64(lsb)
	xlsbInt := int64(xlsb)
	
	up := ((msbInt) << 16 + (lsbInt) << 8 + (xlsbInt)) >> (8 - oss)
	x1 := (int64(ut) - int64(calib.ac6)) * int64(calib.ac5) / (1 << 15)
	x2 := int64(calib.mc) * (1 << 11) / (x1 + int64(calib.md))
	b5 := x1 + x2
	
	b6 := b5 - 4000
	x1 = int64(calib.b2) * (b6 * b6 / (1 << 12)) / (2 << 11)
	x2 = int64(calib.ac2) * b6 / (1 << 11)
	x3 := x1 + x2
	b3 := (((int64(calib.ac1) * 4 + x3) << oss) + 2) / 4
	x1 = int64(calib.ac3) * b6 / (1 << 13)
	x2 = (int64(calib.b1) * (b6 * b6 / (1 << 12))) / (1 << 16)
	x3 = ((x1 + x2) + 2) / 4
	b4 := uint64(calib.ac4) * (uint64(x3 + 32768)) / (1 << 15)
	b7 := (uint64(up) - uint64(b3)) * (50000 >> oss)
	p := int64(0)
	if b7 < (1 << 31) {
		p= int64(b7 * 2 / b4)
	} else {
		p= int64(b7 / b4 * 2)
	}
	x1 = (p >> 8) * (p >> 8)
	x1 = x1 * 3038 / (1 << 16)
	x2 = (-7357 * p) / (1 << 16)
	rslt := float64(p + (x1 + x2 + 3791) / (1 << 4)) / 100.0
	return rslt
}

func calcPressurePascalSealevel(p float64, altitudeMeters float64) float64 {
	return p / math.Pow(1.0-altitudeMeters/44330.0, 5.255)
}
