// Copyright 2020
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBeeLoggerDelLogger(t *testing.T) {
	prefix := "My-Cus"
	l := GetLogger(prefix)
	assert.NotNil(t, l)
	l.Print("hello")

	GetLogger().Print("hello")
	SetPrefix("aaa")
	Info("hello")
}

type mockLogger struct {
	*logWriter
	WriteCost time.Duration `json:"write_cost"` // Simulated log writing time consuming
	writeCnt  int           // Count add 1 when writing log success, just for test result
	lastMsg   *LogMsg       // Last received log message for verification
}

func NewMockLogger() Logger {
	return &mockLogger{
		logWriter: &logWriter{writer: io.Discard},
	}
}

func (m *mockLogger) Init(config string) error {
	return json.Unmarshal([]byte(config), m)
}

func (m *mockLogger) WriteMsg(lm *LogMsg) error {
	m.Lock()
	msg := lm.Msg
	msg += "\n"

	time.Sleep(m.WriteCost)
	if _, err := m.writer.Write([]byte(msg)); err != nil {
		return err
	}

	// Store the last message for verification
	m.lastMsg = lm
	m.writeCnt++
	m.Unlock()
	return nil
}

func (m *mockLogger) GetCnt() int {
	return m.writeCnt
}

func (m *mockLogger) GetLastMsg() *LogMsg {
	m.Lock()
	defer m.Unlock()
	return m.lastMsg
}

func (*mockLogger) Destroy()                    {}
func (*mockLogger) Flush()                      {}
func (*mockLogger) SetFormatter(_ LogFormatter) {}

func TestBeeLogger_AsyncNonBlockWrite(t *testing.T) {
	testCases := []struct {
		name         string
		before       func()
		after        func()
		msgLen       int64
		writeCost    time.Duration
		sendInterval time.Duration
		writeCnt     int
	}{
		{
			// Write log time is less than send log time, no blocking
			name: "mock1",
			after: func() {
				_ = beeLogger.DelLogger("mock1")
			},
			msgLen:       5,
			writeCnt:     10,
			writeCost:    200 * time.Millisecond,
			sendInterval: 300 * time.Millisecond,
		},
		{
			// Write log time is less than send log time, discarded when blocking
			name: "mock2",
			after: func() {
				_ = beeLogger.DelLogger("mock2")
			},
			writeCnt:     5,
			msgLen:       5,
			writeCost:    200 * time.Millisecond,
			sendInterval: 10 * time.Millisecond,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Register(tc.name, NewMockLogger)
			err := beeLogger.SetLogger(tc.name, fmt.Sprintf(`{"write_cost": %d}`, tc.writeCost))
			assert.Nil(t, err)

			l := beeLogger.Async(tc.msgLen)
			l.AsyncNonBlockWrite()

			for i := 0; i < 10; i++ {
				time.Sleep(tc.sendInterval)
				l.Info(fmt.Sprintf("----%d----", i))
			}
			time.Sleep(1 * time.Second)
			assert.Equal(t, tc.writeCnt, l.outputs[0].Logger.(*mockLogger).writeCnt)
			tc.after()
		})
	}
}

func TestBeeLogger_WithFields(t *testing.T) {
	tests := []struct {
		name           string
		fields         map[string]interface{}
		expectedFields map[string]interface{}
		expectNil      bool
	}{
		{
			name:           "nil fields",
			fields:         nil,
			expectedFields: nil,
			expectNil:      true,
		},
		{
			name:           "empty fields",
			fields:         map[string]interface{}{},
			expectedFields: map[string]interface{}{},
			expectNil:      false,
		},
		{
			name: "multiple fields",
			fields: map[string]interface{}{
				"user_id":    123,
				"request_id": "abc123",
				"ip":         "192.168.1.1",
			},
			expectedFields: map[string]interface{}{
				"user_id":    123,
				"request_id": "abc123",
				"ip":         "192.168.1.1",
			},
			expectNil: false,
		},
		{
			name: "mixed types fields",
			fields: map[string]interface{}{
				"int_field":    42,
				"string_field": "hello",
				"bool_field":   true,
				"float_field":  3.14,
			},
			expectedFields: map[string]interface{}{
				"int_field":    42,
				"string_field": "hello",
				"bool_field":   true,
				"float_field":  3.14,
			},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger()
			loggerWithFields := logger.WithFields(tt.fields)
			
			assert.NotNil(t, loggerWithFields)
			
			if tt.expectNil {
				assert.Nil(t, loggerWithFields.fields)
			} else {
				assert.Equal(t, tt.expectedFields, loggerWithFields.fields)
			}
			
			// Verify that original logger is not modified
			assert.Nil(t, logger.fields)
			
			// Test that other properties are copied correctly
			assert.Equal(t, logger.level, loggerWithFields.level)
			assert.Equal(t, logger.prefix, loggerWithFields.prefix)
			assert.Equal(t, len(logger.outputs), len(loggerWithFields.outputs))
		})
	}
}

func TestBeeLogger_WithFields_Logging(t *testing.T) {
	// Register mock logger once before all tests
	Register("mock", NewMockLogger)
	
	tests := []struct {
		name           string
		fields         map[string]interface{}
		logMessage     string
		expectedFields map[string]interface{}
	}{
		{
			name:           "basic fields",
			fields:         map[string]interface{}{"user_id": 456, "action": "login"},
			logMessage:     "User action performed",
			expectedFields: map[string]interface{}{"user_id": 456, "action": "login"},
		},
		{
			name:           "empty fields",
			fields:         map[string]interface{}{},
			logMessage:     "Empty fields test",
			expectedFields: map[string]interface{}{},
		},
		{
			name:           "nil fields",
			fields:         nil,
			logMessage:     "Nil fields test",
			expectedFields: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger()
			err := logger.SetLogger("mock", `{}`)
			assert.Nil(t, err)
			
			// Create logger with fields
			loggerWithFields := logger.WithFields(tt.fields)
			
			// Log a message
			loggerWithFields.Info(tt.logMessage)
			
			// Get the mock logger instance
			mockLog := loggerWithFields.outputs[0].Logger.(*mockLogger)
			lastMsg := mockLog.GetLastMsg()
			
			// Verify the message content
			assert.Equal(t, tt.logMessage, lastMsg.Msg)
			assert.Equal(t, LevelInfo, lastMsg.Level)
			
			// Verify fields in the log message
			if tt.expectedFields == nil {
				assert.Nil(t, lastMsg.Fields)
			} else {
				assert.Equal(t, tt.expectedFields, lastMsg.Fields)
			}
			
			// Clean up
			_ = logger.DelLogger("mock")
		})
	}
}

func TestBeeLogger_WithFields_ConsoleAdapter(t *testing.T) {
	tests := []struct {
		name           string
		fields         map[string]interface{}
		logMessage     string
		expectedFields map[string]interface{}
	}{
		{
			name: "console with fields",
			fields: map[string]interface{}{
				"user_id": 1001,
				"role":    "admin",
			},
			logMessage:     "Console log with fields",
			expectedFields: map[string]interface{}{"user_id": 1001, "role": "admin"},
		},
		{
			name:           "console without fields",
			fields:         nil,
			logMessage:     "Console log without fields",
			expectedFields: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger()
			
			// Use console adapter without colors for consistent testing
			err := logger.SetLogger(AdapterConsole, `{"color": false}`)
			assert.Nil(t, err)
			
			// Create logger with fields
			loggerWithFields := logger.WithFields(tt.fields)
			
			// Log a message - this will output to console but we can't easily capture it
			// The main purpose is to ensure no panics or errors occur
			loggerWithFields.Info(tt.logMessage)
			
			// Verify that fields are set correctly in the logger
			assert.Equal(t, tt.expectedFields, loggerWithFields.fields)
		})
	}
}

func TestBeeLogger_WithFields_FileAdapter(t *testing.T) {
	tests := []struct {
		name           string
		fields         map[string]interface{}
		logMessage     string
		expectedFields map[string]interface{}
	}{
		{
			name: "file with fields",
			fields: map[string]interface{}{
				"session_id": "sess_12345",
				"duration":   1500,
			},
			logMessage:     "File log with fields",
			expectedFields: map[string]interface{}{"session_id": "sess_12345", "duration": 1500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger()
			
			// Use file adapter with a temporary file
			err := logger.SetLogger(AdapterFile, `{"filename": "/tmp/test_withfields.log"}`)
			assert.Nil(t, err)
			
			// Create logger with fields
			loggerWithFields := logger.WithFields(tt.fields)
			
			// Log a message
			loggerWithFields.Info(tt.logMessage)
			
			// Verify that fields are set correctly in the logger
			assert.Equal(t, tt.expectedFields, loggerWithFields.fields)
			
			// Clean up - close and remove the logger
			loggerWithFields.Close()
		})
	}
}

func TestBeeLogger_WithFields_Chaining(t *testing.T) {
	tests := []struct {
		name            string
		firstFields     map[string]interface{}
		secondFields    map[string]interface{}
		expectedFirst   map[string]interface{}
		expectedSecond  map[string]interface{}
	}{
		{
			name:            "chaining different fields",
			firstFields:     map[string]interface{}{"user_id": 123},
			secondFields:    map[string]interface{}{"request_id": "req123"},
			expectedFirst:   map[string]interface{}{"user_id": 123},
			expectedSecond:  map[string]interface{}{"request_id": "req123"},
		},
		{
			name:            "chaining with override",
			firstFields:     map[string]interface{}{"user_id": 123, "role": "user"},
			secondFields:    map[string]interface{}{"user_id": 456}, // Override user_id
			expectedFirst:   map[string]interface{}{"user_id": 123, "role": "user"},
			expectedSecond:  map[string]interface{}{"user_id": 456},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger()
			
			// Test chaining WithFields calls
			logger1 := logger.WithFields(tt.firstFields)
			logger2 := logger1.WithFields(tt.secondFields)
			
			// Verify that each logger has only its own fields (not merged)
			assert.Equal(t, tt.expectedFirst, logger1.fields)
			assert.Equal(t, tt.expectedSecond, logger2.fields)
			
			// Verify that original logger is not affected
			assert.Nil(t, logger.fields)
		})
	}
}

func TestBeeLogger_WithFields_Concurrent(t *testing.T) {
	logger := NewLogger()
	
	// Test concurrent access to WithFields
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("goroutine_%d", i), func(t *testing.T) {
			t.Parallel()
			fields := map[string]interface{}{
				"goroutine": i,
				"timestamp": time.Now().UnixNano(),
			}
			loggerWithFields := logger.WithFields(fields)
			assert.Equal(t, fields, loggerWithFields.fields)
		})
	}
}
