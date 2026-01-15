package main

// Version 版本信息，通过 ldflags 在构建时注入
// 使用 go build -ldflags "-X main.Version=xxx" 注入
var Version = "dev"
