package logger

import "log"

func Info(message ...any) {
	log.Println(append([]any{"info:"}, message...)...)
}

func Error(message ...any) {
	log.Println(append([]any{"error:"}, message...)...)
}

func Fatal(message ...any) {
	log.Fatal(append([]any{"error:"}, message...)...)
}
