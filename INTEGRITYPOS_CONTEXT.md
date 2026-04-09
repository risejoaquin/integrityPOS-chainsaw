# IntegrityPOS Context

Reglas principales:
- domain.Money = int64 centavos
- Arquitectura hexagonal (dominio no importa infraestructura)
- SQLite WAL + transacciones multi-tabla
- HMAC SYNC_PROTOCOL_V1
- stock disponible = stock_actual - stock_reservations activas
