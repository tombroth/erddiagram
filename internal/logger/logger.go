package logger

import "log"

const (
	fatalLabel = "[FATAL] "
	errorLabel = "[ERROR] "
	warnLabel  = "[WARN ] "
	infoLabel  = "[INFO ] "
	debugLabel = "[DEBUG] "
)

// mylog prepends the level string to log.Printf.
// Arguments are handled in the manner of [fmt.Printf].
func mylog(level string, format string, args ...interface{}) {
	log.Printf(level+format, args...)
}

// Fatal calls [log.Fatalf], adding a fatal label.
// Arguments are handled in the manner of [fmt.Printf].
func Fatal(format string, args ...interface{}) {
	log.Fatalf(fatalLabel+format, args...)
}

// Error prints to the standard logger, adding an error label.
// Arguments are handled in the manner of [fmt.Printf].
func Error(format string, args ...interface{}) {
	mylog(errorLabel, format, args...)
}

// Warn prints to the standard logger, adding a warn label.
// Arguments are handled in the manner of [fmt.Printf].
func Warn(format string, args ...interface{}) {
	mylog(warnLabel, format, args...)
}

// Info prints to the standard logger, adding an info label.
// Arguments are handled in the manner of [fmt.Printf].
func Info(format string, args ...interface{}) {
	mylog(infoLabel, format, args...)
}

// Debug prints to the standard logger, adding a debug label.
// Arguments are handled in the manner of [fmt.Printf].
func Debug(format string, args ...interface{}) {
	mylog(debugLabel, format, args...)
}
