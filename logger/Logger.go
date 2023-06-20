package logger

import (
	"log"
)

var DEBUG bool = false

func Init(verbose bool) {
	DEBUG = verbose
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func Info(message ...any) {
	log.Println(append([]any{"info:"}, message...)...)
}

func Error(message ...any) {
	log.Println(append([]any{"error:"}, message...)...)
}

func Fatal(message ...any) {
	log.Fatal(append([]any{"error:"}, message...)...)
}

func Debug(message ...any) {
	if DEBUG {
		log.Println(append([]any{"debug:"}, message...)...)
	}
}
