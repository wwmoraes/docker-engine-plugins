VAULT_TITLE ?= Lab
VAULT_UUID ?= uqbe4ebddaifevsveccesqlanu
ITEM_UUID ?= 7cvueactq2irin4n2gcofqiew4

TOKEN = $(shell cat cmd/op-secret-plugin/token)
OP_SECRET_ENV = cmd/op-secret-plugin/.env.local

curl-item: ${OP_SECRET_ENV}
	@source $< && curl -sSL \
		-H "Authorization: Bearer ${TOKEN}" \
		"$${OP_CONNECT_HOST}/v1/vaults/${VAULT_UUID}/items/${ITEM_UUID}x" \
		| jq .

curl-vault: ${OP_SECRET_ENV}
	@source $< && curl -sSL \
		-H "Authorization: Bearer ${TOKEN}" \
		"$${OP_CONNECT_HOST}/v1/vaults/${VAULT_UUID}" \
		| jq .

curl-vault-by-name: ${OP_SECRET_ENV}
	@source $< && curl -sSL \
		-H "Authorization: Bearer ${TOKEN}" \
		"$${OP_CONNECT_HOST}/v1/vaults?filter=title+eq+%22${VAULT_TITLE}%22" \
		| jq .
