package kiwi

// This file consists of realizations of default formatters.

/* Copyright (c) 2016, Alexander I.Grafov aka Axel
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of kvlog nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

ॐ तारे तुत्तारे तुरे स्व */

import (
	"bytes"
	"strconv"
	"strings"
)

// Formatter represents format of the output.
type Formatter interface {
	// Begin function allows to add prefix string for the output
	// or make some preparations before the output.
	Begin()
	// Pair function called for each key-value pair of the record.
	// It accepts `quoted` for notify when value should be displayed in qoutes.
	// Real look depends of the format so `quoted` hint may be ignored.
	Pair(key, val string, quoted bool)
	// Finish function allows to add suffix string for the output.
	// Also it returns result string for the displaying of the single record.
	// It may be multiline if you wish. Result has no restrictions for you imagination :)
	Finish() []byte
}

type formatLogfmt struct {
	line bytes.Buffer
}

// UseLogfmt says that a sink uses Logfmt format for records output.
func UseLogfmt() *formatLogfmt {
	return new(formatLogfmt)
}

func (f *formatLogfmt) Begin() {
	f.line.Reset()
}

func (f *formatLogfmt) Pair(key, val string, quoted bool) {
	// TODO extend check for all non printable chars, so it need just check for each byte>space
	if strings.ContainsAny(key, " \n\r\t") {
		f.line.WriteString(strconv.Quote(key))
	} else {
		f.line.WriteString(key)
	}
	if val == "" && !quoted {
		f.line.WriteRune(' ')
		return
	}
	f.line.WriteRune('=')
	if quoted {
		f.line.WriteString(strconv.Quote(val))
	} else {
		f.line.WriteString(val)
	}
	f.line.WriteRune(' ')
}

func (f *formatLogfmt) Finish() []byte {
	return f.line.Bytes()
}

type formatJSON struct {
	line bytes.Buffer
}

// UseJSON says that a sink uses JSON (RFC-7159) format for records output.
func UseJSON() *formatJSON {
	return new(formatJSON)
}

func (f *formatJSON) Begin() {
	f.line.Reset()
	f.line.WriteRune('{')
}

func (f *formatJSON) Pair(key, val string, quoted bool) {
	f.line.WriteString(strconv.Quote(key))
	f.line.WriteRune(':')
	if quoted {
		f.line.WriteString(strconv.Quote(val))
	} else {
		f.line.WriteString(val)
	}
	f.line.WriteString(", ")
}

func (f *formatJSON) Finish() []byte {
	f.line.WriteRune('}')
	return f.line.Bytes()
}
