# SYNC_PROTOCOL_V1

Payload pipe-separated:
- receipt_id
- session_id
- total_cents
- created_at
- cajero_id
- terminal_id

Signature: hex(HMAC-SHA256(secret, payload))
