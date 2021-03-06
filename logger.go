package kiwi

// This file consists of Logger related structures and functions.

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
	"strconv"
	"time"
)

type (
	// Logger keeps context and log record. There are many loggers initialized
	// in different places of application. Loggers are not safe for
	// concurrent usage so then you need logger for another goroutine you will need clone existing instance.
	// See Logger.New() method below for details.
	Logger struct {
		contextSrc map[string]interface{}
		context    []pair
		pairs      []pair
	}
	// Stringer is the same as fmt.Stringer
	Stringer interface {
		String() string
	}
	// Valuer allows log data from any custom types in they conform this interface.
	// Also types that conform fmt.Stringer can be used. But as they not have IsQuoted() check
	// they always treated as strings and displayed in quotes.
	Valuer interface {
		Stringer
		IsQuoted() bool
	}
	value struct {
		Strv   string
		Func   interface{}
		Type   uint8
		Quoted bool
	}
	pair struct {
		Key     string
		Val     value
		Deleted bool
	}
)

// New creates a new logger instance.
func New() *Logger {
	return &Logger{contextSrc: make(map[string]interface{})}
}

// New creates copy of logger instance. It copies the context of the old logger
// but skips values of the current record of the old logger.
func (l *Logger) New() *Logger {
	var (
		newContextSrc = make(map[string]interface{})
		newContext    = make([]pair, 0, len(l.context))
	)
	for _, pair := range l.context {
		if !pair.Deleted {
			newContextSrc[pair.Key] = l.contextSrc[pair.Key]
			newContext = append(newContext, pair)
		}
	}
	return &Logger{contextSrc: newContextSrc, context: newContext}
}

// Log is the most common method for flushing previously added key-val pairs to an output.
// After current record is flushed all pairs removed from a record except contextSrc pairs.
func (l *Logger) Log(keyVals ...interface{}) {
	var (
		key    string
		record = make([]pair, 0, len(l.context)+len(l.pairs)+len(keyVals))
	)
	for _, p := range l.context {
		if !p.Deleted {
			if p.Val.Func != nil {
				// evaluate delayed context values
				p.Val.Strv = p.Val.Func.(func() string)()
				record = append(record, pair{p.Key, p.Val, false})
			} else {
				record = append(record, p)
			}
		}
	}
	for _, p := range l.pairs {
		if !p.Deleted {
			if p.Val.Func != nil {
				p.Val.Strv = p.Val.Func.(func() string)()
				record = append(record, pair{p.Key, p.Val, false})
			} else {
				record = append(record, p)
			}
		}
	}
	for i, val := range keyVals {
		if i%2 == 0 {
			key = toRecordKey(val)
			continue
		}
		if value := toRecordValue(val); value.Func != nil {
			value.Strv = value.Func.(func() string)()
			record = append(record, pair{key, value, false})
		} else {
			record = append(record, pair{key, value, false})
		}
	}
	if len(keyVals)%2 == 1 {
		record = append(record, pair{key, value{"", nil, voidVal, false}, false})
	}
	for _, o := range sinks {
		if !o.closed && !o.paused {
			o.In <- &record
		}
	}
	l.pairs = nil
}

// Add a new key-value pairs to the log record. If a key already added then value will be
// updated. If a key already exists in a contextSrc then it will be overridden by a new
// value for a current record only. After flushing a record with Log() old context value
// will be restored.
func (l *Logger) Add(keyVals ...interface{}) *Logger {
	var (
		key string
	)
	// key=val pairs
	for i, val := range keyVals {
		if i%2 == 0 {
			key = toRecordKey(val)
			continue
		}
		l.pairs = append(l.pairs, pair{key, toRecordValue(val), false})
	}
	//  add a key without value for odd number for key-val pairs
	if len(keyVals)%2 == 1 {
		l.pairs = append(l.pairs, pair{key, value{"", nil, voidVal, false}, false})
	}
	return l
}

// With defines a context for the logger. The context overrides pairs in the record.
func (l *Logger) With(keyVals ...interface{}) *Logger {
	var (
		key string
	)
	// key=val pairs
	for i, val := range keyVals {
		if i%2 == 0 {
			key = toRecordKey(val)
			continue
		}
		// keep context keys unique
		if _, ok := l.contextSrc[key]; ok {
			for i, pair := range l.context {
				if pair.Key == key {
					l.context[i].Val = toRecordValue(val)
					break
				}
			}
		} else {
			l.context = append(l.context, pair{key, toRecordValue(val), false})
		}
		l.contextSrc[key] = val
	}
	// add a key without value for odd number for key-val pairs
	if len(keyVals)%2 == 1 {
		if _, ok := l.contextSrc[key]; ok {
			for i, pair := range l.context {
				if pair.Key == key {
					l.context[i].Val = value{"", nil, voidVal, false}
					break
				}
			}
		} else {
			l.context = append(l.context, pair{key, value{"", nil, voidVal, false}, false})
		}
		l.contextSrc[key] = nil
	}
	return l
}

// Without drops some keys from a context for the logger.
func (l *Logger) Without(keys ...string) *Logger {
	for _, key := range keys {
		ckey := toRecordKey(key)
		if _, ok := l.contextSrc[key]; ok {
			delete(l.contextSrc, key)
			for i, pair := range l.context {
				if pair.Key == ckey {
					l.context[i].Deleted = true
					break
				}
			}
		}
	}
	return l
}

// WithTimestamp adds "timestamp" field to the context.
func (l *Logger) WithTimestamp(format string) *Logger {
	l.contextSrc["timestamp"] = func() string { return time.Now().Format(format) }
	l.context = append(l.context, pair{"timestamp", value{"", func() string { return time.Now().Format(format) }, stringVal, true}, false})
	return l
}

// Reset logger values added after last Log() call. It keeps context untouched.
func (l *Logger) Reset() *Logger {
	l.pairs = nil
	return l
}

// ResetContext resets the context of the logger.
func (l *Logger) ResetContext() *Logger {
	l.contextSrc = make(map[string]interface{})
	l.context = nil
	return l
}

// GetContext returns copy of the context saved in the logger.
func (l *Logger) GetContext() map[string]interface{} {
	var contextSrcCopy = make(map[string]interface{})
	for _, pair := range l.context {
		if !pair.Deleted {
			contextSrcCopy[pair.Key] = l.contextSrc[pair.Key]
		}
	}
	return contextSrcCopy
}

// GetContextValue returns single context value for the key.
// It can return values deleted from the context.
func (l *Logger) GetContextValue(key string) interface{} {
	value := l.contextSrc[key]
	return value
}

// AddString adds key with a string value to a record.
func (l *Logger) AddString(key string, val string) *Logger {
	l.pairs = append(l.pairs, pair{key, value{val, nil, stringVal, true}, false})
	return l
}

// AddStringer adds key with a string value to a record.
// It using Stringer interface that is the same as fmt.Stringer.
func (l *Logger) AddStringer(key string, val Stringer) *Logger {
	l.pairs = append(l.pairs, pair{key, value{val.String(), nil, stringVal, true}, false})
	return l
}

// AddInt adds key with a int value to a record.
func (l *Logger) AddInt(key string, val int) *Logger {
	l.pairs = append(l.pairs, pair{key, value{strconv.Itoa(val), nil, integerVal, true}, false})
	return l
}

// AddInt64 adds key with a int64 value to a record.
func (l *Logger) AddInt64(key string, val int64) *Logger {
	l.pairs = append(l.pairs, pair{key, value{strconv.FormatInt(val, 10), nil, integerVal, true}, false})
	return l
}

// AddUint64 adds key with a uint64 value to a record.
func (l *Logger) AddUint64(key string, val uint64) *Logger {
	l.pairs = append(l.pairs, pair{key, value{strconv.FormatUint(val, 10), nil, integerVal, true}, false})
	return l
}

// AddFloat64 adds key with a float64 value to a record.
func (l *Logger) AddFloat64(key string, val float64) *Logger {
	l.pairs = append(l.pairs, pair{key, value{strconv.FormatFloat(val, 'e', -1, 64), nil, floatVal, true}, false})
	return l
}

// AddBool adds key with a boolean value to a record.
func (l *Logger) AddBool(key string, val bool) *Logger {
	if val {
		l.pairs = append(l.pairs, pair{key, value{"true", nil, booleanVal, false}, false})
	} else {
		l.pairs = append(l.pairs, pair{key, value{"false", nil, booleanVal, false}, false})
	}
	return l
}

// AddPairs adds pairs to a record. Each pair represent a key with with a value of strict type.
func (l *Logger) AddPairs(pairs ...pair) *Logger {
	l.pairs = append(l.pairs, pairs...)
	return l
}
