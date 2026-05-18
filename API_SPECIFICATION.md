# API_SPECIFICATION.md - IntegrityPOS Backend

**Version:** 1.0  
**Base URL:** `http://localhost:8080/api/v1`  
**Status Codes:** Standard HTTP (200, 400, 401, 403, 404, 409, 500)

---

## Authentication Endpoints

### POST /auth/login
**Description:** Authenticate user with username and PIN  
**Authentication:** None (public endpoint)

**Request:**
```json
{
  "username": "string (required)",
  "password": "string (required, PIN)"
}
```

**Response 200 - Success:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": {
    "id": 1,
    "username": "cashier1",
    "email": "cashier1@store.com",
    "role": "cashier"
  }
}
```

**Response 401 - Invalid Credentials:**
```json
{
  "error": "invalid_credentials",
  "message": "Username or password is incorrect"
}
```

**Response 400 - Validation Error:**
```json
{
  "error": "validation_error",
  "message": "Username and password are required"
}
```

---

## Shift Endpoints (Protected - Requires JWT)

### POST /shifts/open
**Description:** Open a new shift for the current user  
**Authentication:** Required (Bearer token)  
**Authorization:** cashier, admin, manager

**Request:**
```json
{
  "open_balance": 10000
}
```

Where:
- `open_balance`: Initial cash balance in cents (required, integer, ≥0)

**Response 200 - Success:**
```json
{
  "id": 5,
  "user_id": 1,
  "opened_at": "2026-05-07T08:00:00Z",
  "closed_at": null,
  "open_balance": 10000,
  "close_balance": null,
  "notes": "",
  "created_at": "2026-05-07T08:00:00Z",
  "updated_at": "2026-05-07T08:00:00Z"
}
```

**Response 400 - Invalid Data:**
```json
{
  "error": "invalid_data",
  "message": "open_balance must be a non-negative integer"
}
```

**Response 409 - Already Has Open Shift:**
```json
{
  "error": "shift_already_open",
  "message": "Cannot open a new shift while one is already open. Close the current shift first."
}
```

**Response 401 - Unauthorized:**
```json
{
  "error": "unauthorized",
  "message": "Invalid or missing authentication token"
}
```

---

### POST /shifts/close
**Description:** Close the current open shift  
**Authentication:** Required (Bearer token)  
**Authorization:** cashier, admin, manager

**Request:**
```json
{
  "close_balance": 11500
}
```

Where:
- `close_balance`: Final cash balance in cents (required, integer, ≥0)

**Response 200 - Success:**
```json
{
  "id": 5,
  "user_id": 1,
  "opened_at": "2026-05-07T08:00:00Z",
  "closed_at": "2026-05-07T16:00:00Z",
  "open_balance": 10000,
  "close_balance": 11500,
  "notes": "",
  "created_at": "2026-05-07T08:00:00Z",
  "updated_at": "2026-05-07T16:00:00Z"
}
```

**Response 404 - No Open Shift:**
```json
{
  "error": "no_open_shift",
  "message": "No open shift found for the current user"
}
```

**Response 400 - Invalid Data:**
```json
{
  "error": "invalid_data",
  "message": "close_balance must be a non-negative integer"
}
```

---

### GET /shifts/current
**Description:** Get the current open shift for the authenticated user  
**Authentication:** Required (Bearer token)  
**Authorization:** cashier, admin, manager

**Response 200 - Success:**
```json
{
  "id": 5,
  "user_id": 1,
  "opened_at": "2026-05-07T08:00:00Z",
  "closed_at": null,
  "open_balance": 10000,
  "close_balance": null,
  "notes": "",
  "created_at": "2026-05-07T08:00:00Z",
  "updated_at": "2026-05-07T08:00:00Z"
}
```

**Response 404 - No Open Shift:**
```json
{
  "error": "no_open_shift",
  "message": "No open shift found for the current user"
}
```

---

### GET /shifts/:id
**Description:** Get a specific shift by ID  
**Authentication:** Required (Bearer token)  
**Authorization:** cashier (own shifts), admin, manager

**Response 200 - Success:**
```json
{
  "id": 5,
  "user_id": 1,
  "opened_at": "2026-05-07T08:00:00Z",
  "closed_at": "2026-05-07T16:00:00Z",
  "open_balance": 10000,
  "close_balance": 11500,
  "notes": "",
  "created_at": "2026-05-07T08:00:00Z",
  "updated_at": "2026-05-07T16:00:00Z"
}
```

**Response 404 - Not Found:**
```json
{
  "error": "not_found",
  "message": "Shift not found"
}
```

**Response 403 - Forbidden:**
```json
{
  "error": "forbidden",
  "message": "You don't have permission to view this shift"
}
```

---

## Error Response Format

All error responses follow this format:

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {} // Optional additional details
}
```

**Common Error Codes:**
- `validation_error` - Request validation failed
- `invalid_credentials` - Auth failed
- `unauthorized` - Missing/invalid token
- `forbidden` - Insufficient permissions
- `not_found` - Resource not found
- `conflict` - Resource conflict (e.g., duplicate)
- `internal_error` - Server error

---

## Authentication Header

All protected endpoints require:

```
Authorization: Bearer <access_token>
```

Example:
```
GET /api/v1/shifts/current HTTP/1.1
Host: localhost:8080
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

---

## Data Types

### Monetary Values
- **Type:** `integer`
- **Unit:** Cents
- **Example:** $19.99 = 1999
- **Range:** 0 to 9223372036854775807 (max int64)
- **Note:** Never float, always integer

### Timestamps
- **Format:** RFC3339 (ISO 8601)
- **Example:** "2026-05-07T08:00:00Z"
- **Timezone:** UTC

### User Roles
- `cashier` - Standard user
- `admin` - Full system access
- `manager` - Management access

### Payment Methods
- `cash` - Cash payment
- `card` - Card payment
- `check` - Check payment

---

## Rate Limiting

Currently not implemented. To be added in future versions.

---

## Versioning

API version is in the URL path: `/api/v1`

Future versions will use `/api/v2`, `/api/v3`, etc.

---

## Notes

1. **Monetary Values:** All prices, costs, and balances are integers representing cents
2. **Immutability:** Sales records cannot be deleted, only voided
3. **Time Tracking:** All timestamps are server-generated and in UTC
4. **Atomic Operations:** Shift open/close are atomic (all or nothing)
5. **Local-First:** This API works offline; sync to cloud happens asynchronously
