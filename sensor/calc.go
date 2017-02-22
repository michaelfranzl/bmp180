// Copyright 2017 Michael Franzl. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The following calculations are based on a paper hosted on:
// http://wmrx00.sourceforge.net/Arduino/BMP085-Calcs.pdf

package sensor

import (
	"math"
)

func calcConstants(calib calibration) constants {
	var cnsts constants

	cnsts.c5 = math.Pow(2, -15) / 160 * float64(calib.ac5)
	cnsts.c6 = float64(calib.ac6)
	cnsts.mc = 2048.0 / (160 * 160) * float64(calib.mc)
	cnsts.md = float64(calib.md) / 160

	c3 := 160 * math.Pow(2, -15) * float64(calib.ac3)
	c4 := 0.001 * math.Pow(2, -15) * float64(calib.ac4)
	b1 := 160 * 160 * math.Pow(2, -30) * float64(calib.b1)

	cnsts.x0 = float64(calib.ac1)
	cnsts.x1 = 160 * math.Pow(2, -13) * float64(calib.ac2)
	cnsts.x2 = 160 * 160 * math.Pow(2, -25) * float64(calib.b2)

	cnsts.y0 = c4 * 32768
	cnsts.y1 = c4 * c3
	cnsts.y2 = c4 * b1

	cnsts.p0 = (3791 - 8) / 1600.0
	cnsts.p1 = 1 - 7357*math.Pow(2, -20)
	cnsts.p2 = 3038 * 100 * math.Pow(2, -36)

	return cnsts
}

func calcTempCelsius(tu uint16, cnsts constants) float64 {
	alpha := cnsts.c5 * (float64(tu) - cnsts.c6)
	t := alpha + (cnsts.mc / (alpha + cnsts.md))
	return t
}

func calcPressurePascal(tempCelsius float64, msb, lsb, xlsb byte, cnsts constants) float64 {
	s := tempCelsius - 25
	x := cnsts.x2*s*s + cnsts.x1*s + cnsts.x0
	y := cnsts.y2*s*s + cnsts.y1*s + cnsts.y0

	msbInt := uint8(msb)
	lsbInt := uint8(lsb)
	xlsbInt := uint8(xlsb)

	pu := float64(msbInt)*256 + float64(lsbInt) + float64(xlsbInt)/256

	z := (pu - x) / y
	p := cnsts.p2*z*z + cnsts.p1*z + cnsts.p0

	return p
}

func calcPressurePascalSealevel(p float64, altitudeMeters float64) float64 {
	return p / math.Pow(1.0-altitudeMeters/44330.0, 5.255)
}
