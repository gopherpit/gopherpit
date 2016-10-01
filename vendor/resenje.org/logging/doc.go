// Package logging is async logger library. It is based on idea that
// log messages are created when client wants to log something, but
// actual processing is done is designated goroutine. This means that
// main application will not be blocked while logging is performed with
// IO operations (writing to file, stdout/stderr, to socket).
//
// However, be warned that if your application panics and quits, it is
// possible that some log messages will be lost, since they might not
// be processed.
//
// Most simple way of using this module is by just importing it and
// start logging. Default logger (that writes messages to stderr)
// is created by default.
//
// Example:
//
//     import "resenje.org/logging"
//     func main() {
//         logging.Info("Some message")
//         logging.WaitForAllUnprocessedRecords()
//     }
//
// Since this is async logger, last line in main function is needed
// because program will end before logger has a chance to process message.
package logging
