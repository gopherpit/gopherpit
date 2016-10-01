package logging

import (
	"bytes"
	"encoding/json"
	"errors"
)

// ErrInvalidLevel is error returned if no valid log level can be decoded.
var ErrInvalidLevel = errors.New("invalid log level")

// Levels of logging supported by library.
// They are ordered in descending order or imporance.
const (
	EMERGENCY Level = iota
	ALERT
	CRITICAL
	ERROR
	WARNING
	NOTICE
	INFO
	DEBUG
)

const (
	stopped uint8 = iota
	paused
	running
)

// Level represents log level for log message.
type Level int8

// String returns stirng representation of log level.
func (level Level) String() string {
	switch level {
	case EMERGENCY:
		return "EMERGENCY"
	case ALERT:
		return "ALERT"
	case CRITICAL:
		return "CRITICAL"
	case ERROR:
		return "ERROR"
	case WARNING:
		return "WARNING"
	case NOTICE:
		return "NOTICE"
	case INFO:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// MarshalJSON is implementation of json.Marshaler interface, will be used when
// log level is serialized to json.
func (level Level) MarshalJSON() ([]byte, error) {
	return json.Marshal(level.String())
}

// UnmarshalJSON implements json.Unamrshaler interface.
func (level *Level) UnmarshalJSON(data []byte) error {
	switch string(bytes.ToUpper(data)) {
	case `"EMERGENCY"`, "0":
		*level = EMERGENCY
	case `"ALERT"`, "1":
		*level = ALERT
	case `"CRITICAL"`, "2":
		*level = CRITICAL
	case `"ERROR"`, "3":
		*level = ERROR
	case `"WARNING"`, "4":
		*level = WARNING
	case `"NOTICE"`, "5":
		*level = NOTICE
	case `"INFO"`, "6":
		*level = INFO
	case `"DEBUG"`, "7":
		*level = DEBUG
	default:
		return ErrInvalidLevel
	}
	return nil
}

// MarshalText is implementation of encoding.TextMarshaler interface,
// will be used when log level is serialized to text.
func (level Level) MarshalText() ([]byte, error) {
	return []byte(level.String()), nil
}

// UnmarshalText implements encoding.TextUnamrshaler interface.
func (level *Level) UnmarshalText(data []byte) error {
	switch string(bytes.ToUpper(data)) {
	case "EMERGENCY", "0":
		*level = EMERGENCY
	case "ALERT", "1":
		*level = ALERT
	case "CRITICAL", "2":
		*level = CRITICAL
	case "ERROR", "3":
		*level = ERROR
	case "WARNING", "4":
		*level = WARNING
	case "NOTICE", "5":
		*level = NOTICE
	case "INFO", "6":
		*level = INFO
	case "DEBUG", "7":
		*level = DEBUG
	default:
		return ErrInvalidLevel
	}
	return nil
}
