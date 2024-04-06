source /koxtoolchain/refs/x-compile.sh kobo env bare
CGO_ENABLED=1 GOARCH=arm GOOS=linux CC=/home/build/x-tools/arm-kobo-linux-gnueabihf/bin/arm-kobo-linux-gnueabihf-gcc CXX=/home/build/x-tools/arm-kobo-linux-gnueabihf/bin/arm-kobo-linux-gnueabihf-g++ go build touch.go
