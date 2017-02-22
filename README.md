# golang package for BMP180 sensor

[![Build Status](https://travis-ci.org/michaelfranzl/bmp180.svg?branch=master)](https://travis-ci.org/michaelfranzl/bmp180)

Package `sensor` provides an interface to the BMP180 family of environmental (temperature, pressure) sensors attached to an I2C interface.

Package `i2c_stub` provides an emulated BMP180 device that can be used to test functionality of the `sensor` package when no physical device is attached or available.

Pressure and temperature calculations are based on a paper called ["Bosch BMP085 Barometer Floating Point Pressure Calculations"](http://wmrx00.sourceforge.net/Arduino/BMP085-Calcs.pdf).

See `sensor/sensor_test.go` for a working example.

Copyright 2017 Michael Franzl. All rights reserved.

Use of this source code is governed by a BSD-style license that can be found in the LICENSE file.
