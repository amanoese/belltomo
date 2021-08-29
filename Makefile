
build:
	tinygo build -target=arduino-nano33 -o ./test.hex ./main.go

flash:
	tinygo flash -target=arduino-nano33 ./main.go
