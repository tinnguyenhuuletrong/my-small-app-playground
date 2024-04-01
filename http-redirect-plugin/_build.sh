go build -buildmode=plugin -o plugin.so log-plugin/plugin.go
mv ./plugin.so ./data/log-plugin.so