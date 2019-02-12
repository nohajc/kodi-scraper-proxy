CXX=g++
CXXFLAGS=-fPIC -std=c++17
LDFLAGS=-L../../pkg/libfilter -lfilter

curl_hook.so: curl_hook.o
	g++ -shared -fPIC $< -o $@ $(LDFLAGS)
