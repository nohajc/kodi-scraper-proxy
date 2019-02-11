CXX=g++
CXXFLAGS=-fPIC -std=c++17

curl_hook.so: curl_hook.o
	g++ -shared -fPIC $< -o $@
