module example.com/supply-chain-demo

go 1.21

// 意図的に脆弱なバージョンを使用 (govulncheck のデモ用)
require golang.org/x/crypto v0.13.0

require golang.org/x/sys v0.12.0 // indirect
