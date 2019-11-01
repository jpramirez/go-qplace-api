TIME=$(shell date +"%Y%m%d.%H%M%S")
VERSION=0.1.1-alpha-0.8
BINARY_NAME=go-qplace-api-go.v1

BINARY_NAME_SERVER=go-qplace-api-server.v1


BUILD_FOLDER  = $(shell pwd)/build

FLAGS_LINUX   = CGO_LDFLAGS="-L./LIB -Wl,-rpath -Wl,\$ORIGIN/LIB" CGO_ENABLED=1 GOOS=linux GOARCH=amd64  
FLAGS_DARWIN  = OSXCROSS_NO_INCLUDE_PATH_WARNINGS=1 MACOSX_DEPLOYMENT_TARGET=10.6 CC=o64-clang CXX=o64-clang++ CGO_ENABLED=0
FLAGS_FREEBSD = GOOS=freebsd GOARCH=amd64 CGO_ENABLED=1
FLAGS_WINDOWS = GOOS=windows GOARCH=386 CC=i686-w64-mingw32-gcc CGO_ENABLED=1 

GOFLAGS_WINDOWS = -ldflags -H=windowsgui

check-env:
	@mkdir -p $(BUILD_FOLDER)/dist/linux/bin
	@mkdir -p $(BUILD_FOLDER)/dist/windows/bin
	@mkdir -p $(BUILD_FOLDER)/dist/arm/bin
	@mkdir -p $(BUILD_FOLDER)/dist/osx/bin
	cp -R config $(BUILD_FOLDER)/dist/linux/
	cp -R config $(BUILD_FOLDER)/dist/windows/
	cp -R config $(BUILD_FOLDER)/dist/arm/
	cp -R config $(BUILD_FOLDER)/dist/osx/
	cp -R extras $(BUILD_FOLDER)/dist/linux/
	cp -R templates $(BUILD_FOLDER)/dist/linux/




## Linting
lint:
	@echo "[lint] Running linter on codebase"
	@golint ./...


getdeps:
	./getDeps.sh




versioning:
	./version.sh ${VERSION} ${TIME}


compile/webresources: 
	cd web/ && go-bindata -nometadata -o statics/bindata.go -pkg=statics -ignore=bindata.go js/*.js statics/index.html statics/css/* statics/css/themes/*/* statics/fonts/* statics/img/* statics/js/*

build/weblayer-linux:
	cd cmd/WebServer && ${FLAGS_LINUX} go build -o ${BUILD_FOLDER}/dist/linux/bin/${BINARY_NAME_SERVER} .


run/dev:
	cd build/dist/linux && bin/${BINARY_NAME_SERVER} --config config/config.json

build/dev: check-env build/weblayer-linux run/dev

clean:
	rm -Rvf build/dist/