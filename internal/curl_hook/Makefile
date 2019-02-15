CXX := g++
CXXFLAGS := ${CXXFLAGS} -fPIC -std=c++17
LDFLAGS := -L. -lbridge

curl_hook.so: curl_hook.o
	g++ -shared -fPIC $< -o $@ $(LDFLAGS)

curl_hook.o: libbridge.h

libbridge.h: libbridge.so

libbridge.so: libbridge.go
	go build -buildmode=c-shared -o $@ $<

install: curl_hook.so libbridge.so
	mkdir -p ../../bin
	cp $^ ../../bin
	cp ../../cmd/install_hook/install_hook.sh ../../bin
