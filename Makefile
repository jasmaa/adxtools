
APP_NAME=adxtools

.PHONY: all clean

all:
	go build -o $(APP_NAME).exe

clean:
	rm $(APP_NAME).exe
