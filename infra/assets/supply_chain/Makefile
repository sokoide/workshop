IMAGE ?= supply-chain-demo
TAG  ?= latest

.PHONY: vuln scan sbom sign verify all clean

# ソースコードの脆弱性スキャン
vuln:
	govulncheck ./...

# Docker イメージのビルド
build:
	docker build -t $(IMAGE):$(TAG) .

# コンテナイメージの脆弱性スキャン
scan:
	trivy image --severity HIGH,CRITICAL $(IMAGE):$(TAG)

# SBOM 生成とスキャン
sbom:
	syft $(IMAGE):$(TAG) -o table
	syft $(IMAGE):$(TAG) -o spdx-json > sbom.spdx.json
	grype sbom:sbom.spdx.json --fail-on high

# イメージ署名 (鍵ペアが必要)
sign: build
	cosign sign --key cosign.key $(IMAGE):$(TAG)

# 署名検証
verify:
	cosign verify --key cosign.pub $(IMAGE):$(TAG)

# 全セキュリティチェック実行
all: vuln build scan sbom
	@echo "All security checks completed"

clean:
	rm -f sbom.spdx.json cosign.key cosign.pub
