module github.com/sokoide/workshop/infra/assets/supply_chain

go 1.21

// 意図的に脆弱なバージョンを使用 (govulncheck のデモ用)
require golang.org/x/crypto v0.13.0

require golang.org/x/sys v0.12.0 // indirect
