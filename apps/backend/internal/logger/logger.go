package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

var output = log.New(os.Stderr, "", 0)

func Info(message string, args ...any)  { write("INFO", "\x1b[36m", message, args...) }
func Warn(message string, args ...any)  { write("WARN", "\x1b[33m", message, args...) }
func Error(message string, args ...any) { write("ERROR", "\x1b[31m", message, args...) }

func write(level, color, message string, args ...any) {
	output.Printf("%s%s [%s]\x1b[0m %s", color+time.Now().Format("15:04:05"), color, level, fmt.Sprintf(message, args...))
}
