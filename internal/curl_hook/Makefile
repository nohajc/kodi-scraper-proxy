CXX=g++
CXXFLAGS=-fPIC -std=c++17
LDFLAGS=-L bridge -lbridge

curl_hook.so: curl_hook.o
	g++ -shared -fPIC $< -o $@ $(LDFLAGS)

curl_hook.o: bridge/libbridge.h

bridge/libbridge.h: libbridge.a

libbridge.a: bridge/libbridge.go
	(cd bridge && go build -buildmode=c-archive -o $@)
