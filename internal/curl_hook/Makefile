CXX=g++
CXXFLAGS=-fPIC -std=c++17
FILTER_PATH=../../pkg/libfilter
LDFLAGS=-L$(FILTER_PATH) -lfilter

curl_hook.so: curl_hook.o
	g++ -shared -fPIC $< -o $@ $(LDFLAGS)

curl_hook.o: libfilter.a

libfilter.a: $(FILTER_PATH)/libfilter.go
	(cd $(FILTER_PATH) && go build -buildmode=c-archive -o $@)
