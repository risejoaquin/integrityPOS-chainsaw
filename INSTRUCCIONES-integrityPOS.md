# INSTRUCCIONES DE DESARROLLO BACKEND - INTEGRITY POS

**Fecha de Referencia:** Mayo de 2026
**Rol del Agente (Cline):** Principal Software Engineer experto en Go (Golang), Arquitectura Hexagonal y entornos Local-First.

## 1. Contexto y Directrices Fundamentales
Desarrollarás el backend completo del sistema **integrityPOS**, que se ejecutará como un *sidecar* en Go dentro de una Raspberry Pi 4 o PC nativo (ambientes Windows/Linux).

*   **Motor de Base de Datos:** Se utilizará **SQLite en modo WAL** incrustado (embedded) exclusivamente. Para la fase de desarrollo y pruebas usaremos la base local, y en producción operará completamente offline priorizando la velocidad y resiliencia. La sincronización a la nube (Supabase) es un proceso en segundo plano asíncrono.
*   **Tipos de Datos Críticos:** Todo manejo de dinero **DEBE** ser procesado obligatoriamente como entero (`int64` representando centavos). PROHIBIDO el uso de floats (`float32`, `float64`) para montos monetarios a fin de prevenir errores de coma flotante.
*   **Versiones y Stack Tecnológico (Mayo 2026):** Usa código optimizado para **Go 1.22**. Emplea librerías maduras y consolidadas (ej. `github.com/mattn/go-sqlite3` para SQLite con soporte CGO, `github.com/gin-gonic/gin` para el enrutamiento HTTP rápido, `golang.org/x/crypto/bcrypt` para hashing, y `github.com/golang-jwt/jwt/v5` para tokens).

## 2. Buenas Prácticas de Control de Versiones (Git)
Cline, deberás gestionar el repositorio local aplicando estas normas:
*   **Semantic Commits:** Tus descripciones de guardado deben usar el estándar semántico (`feat:`, `fix:`, `chore:`, `refactor:`, `test:`).
*   **Commits Atómicos:** Realiza *commits* independientes y aislados por capa en lugar de un macro-commit. (Ej. `feat(core): establecer entidades de negocio para transacciones POS`).
*   **Archivos Ignorados:** Antes de cualquier línea de código, verifica que exista un `.gitignore` correcto que pase por alto los compilados (`/bin`, `*.exe`), variables sensibles (`.env`), la base de datos real (`*.sqlite`, `*.db-*`) y la caché del sistema.

## 3. Plan de Implementación de Backend (Arquitectura Hexagonal)
Debes generar todo el código de manera estructurada, garantizando que el diseño compile sin errores en cada paso antes de saltar a la siguiente capa.

### Paso 3.1: Dominio (Core - Entities)
Crea las entidades anémicas puras de Go en `internal/core/domain`. Configura con *struct tags* tanto de JSON como de bdd (si es necesario para ORM ligero / `sqlx`).
*   Entidades base: `User`, `Shift`, `Product`, `Sale`, `SaleItem`, `SyncLog`. Estrictamente alineadas con los esquemas de `DATABASE_SCHEMA.sql`.

### Paso 3.2: Puertos (Interfaces de I/O)
En `internal/core/ports`, plantea los contratos primarios y secundarios.
*   **Driven (Outbound):** `UserRepository`, `ShiftRepository`, `ProductRepository`, `SaleRepository` (para interactuar con SQLite), y `HardwareLockService` (para leer estado hardware).
*   **Driving (Inbound):** Operaciones expuestas por casos de uso (`AuthUseCase`, `PosUseCase`, `InventoryUseCase`).

### Paso 3.3: Lógica Aplicativa (Services)
Escribe los procesos de validación de negocio en `internal/core/services`.
*   **Transaccionalidad Atómica:** El motor de ventas debe manejar la persistencia múltiple (descontar stock, insertar venta principal, insertar detalle, guardar log de auditoría/sincronización). Si todo pasa, hace *commit*. Si un producto marca error, hace *rollback*.
*   **Inmutabilidad:** Las ventas y *logs* bajo ningún motivo pueden tener funciones de actualización (`UPDATE`) o borrado (`DELETE`).

### Paso 3.4: Adaptadores (Infraestructura)
Implementa las interfaces (puertos).
*   **Driver SQLite (`internal/adapters/repositories/sqlite`):** Conexión robusta asegurando inyectar explícitamente `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;` al abrir la DB.
*   **Driver de Hardware (`internal/adapters/hardware`):** Ejecución nativa (usando el paquete `os/exec` o syscalls) para hacer lectura transparente del UUID/Serial de la Motherboard e interacciones como la de `kick-drawer` para el cajón y `print-ticket` ESC/POS.
*   **Capa HTTP (`internal/adapters/handlers`):** Implementa Gin. Instala el Middleware de HWID que intercepte y valide la firma HMAC para verificar que la DB y el disco están corriendo en la Motherboard autorizada.

### Paso 3.5: Punto de Entrada e Inyección (`cmd/api/main.go`)
Conecta todo. Lee variables de entorno, inicializa el *pool* de la base de datos, inyecta repositorios en los servicios, mapea servicios en el handler e inicia la escucha sobre el puerto (por defecto `8080`).

## 4. Orquestación DevOps Local (Docker Compose Estable)
Para no tener errores de librerías ni de dependencias nativas durante la programación diaria y las pruebas cruzadas, deberás proveer un `docker-compose.yml`. 

Ya que usamos `go-sqlite3`, se requiere habilitar **CGO**, por tanto necesitamos `gcc/musl-dev`. Genera en la capa superior del backend el siguiente archivo exacto (o adáptalo si fuera necesario manteniendo las versiones ancladas de Alpine 3.19+ y Go 1.22):

```yaml
version: '3.8'

services:
  backend-dev:
    image: golang:1.22-alpine3.19
    container_name: integritypos-backend
    working_dir: /app
    volumes:
      - .:/app
      # Volumen anónimo para cachear dependencias y que compile ultra rápido
      - go_pkg_cache:/go/pkg
    ports:
      - "8080:8080"
    environment:
      # CRÍTICO: CGO debe estar en 1 para compilar el driver driver SQLite nativo
      - CGO_ENABLED=1
      - PORT=8080
      - GIN_MODE=debug
      - JWT_SECRET=dev_integrity_secret_2026
      - HWID_SALT=dev_integrity_salt_2026
      # Simulador de HWID activado, clave para esquivar el bloqueo del hardware 
      # subyacente que Docker no puede leer de forma directa
      - MOCK_HWID=true 
    command: >
      sh -c "apk add --no-cache gcc musl-dev &&
             go mod tidy &&
             go run ./cmd/api/main.go"

volumes:
  go_pkg_cache:
```

## 5. Verificación Final de Compilación
Antes de dar por concluida la creación del backend, Cline debe:
1. Emplear el comando de ejecución `go test ./...` y verificar que el tipado y paquetes anidados cuadren perfectamente sin *panics* ni "import cycles".
2. Revisar que no haya importaciones que no son usadas (`go mod tidy`).
3. Verificar la consistencia atómica del módulo de SQLite.
