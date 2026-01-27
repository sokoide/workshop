.PHONY: vault-up vault-down vault-logs vault-status

vault-up:
	podman run -d --name workshop-vault \
		-p 8200:8200 \
		-e 'VAULT_DEV_ROOT_TOKEN_ID=dev-workshop-token' \
		-e 'VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200' \
		docker.io/hashicorp/vault:latest

vault-down:
	podman stop workshop-vault || true
	podman rm workshop-vault || true

vault-logs:
	podman logs -f workshop-vault

vault-status:
	@podman ps | grep workshop-vault || echo "Vault not running"
