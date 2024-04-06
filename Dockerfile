FROM golang:latest

RUN useradd -ms /bin/bash build

WORKDIR /

RUN apt-get update
RUN apt-get install -y build-essential autoconf automake bison flex gawk libtool libtool-bin libncurses-dev curl file git gperf help2man texinfo zip unzip wget git

RUN git clone https://github.com/koreader/koxtoolchain.git
RUN chown -R build koxtoolchain

WORKDIR /koxtoolchain

USER build
RUN ["./gen-tc.sh", "kobo"]
ENV CGO_ENABLED=1
ENV GOARCH=arm
ENV CC=/home/build/x-tools/arm-kobo-linux-gnueabihf/bin/arm-kobo-linux-gnueabihf-gcc
ENV CXX=/home/build/x-tools/arm-kobo-linux-gnueabihf/bin/arm-kobo-linux-gnueabihf-g++
ENTRYPOINT "source /koxtoolchain/refs/x-compile.sh kobo env bare"
