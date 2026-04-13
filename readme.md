
# Notification API Center
A robust API written in go, designed to centralize push notification management

## Features

*   **Security:** JWT Auth.
*   **Scalability:** Rate Limiting.
*   **Web push:** Subscription and bulk notification send.
*   **Modularity:** Clear responsability separation between domains (Applications, Auth, Users, Notifications).

## Instalation and configuration

```bash
git clone https://github.com/Angel-del-dev/push-notification-center.git push-notification-center
cd push-notification-center
go mod download
cp .env-example .env
```

```sql
--psql
|> insert into applications(name, description) values ('...', '...');
|> insert into applications_keys(application, password) values ('...', '...');
```

## Structure and execution

### Dependencies
*   `github.com/gofiber/fiber/v3`: Go Web framework.
*   `github.com/jackc/pgx/v5`: PostgreSQL driver.
*   `golang.org/x/crypto/bcrypt`: Secure hash creation/validation.

### Server startup

```bash
cd scripts
./runbuild.sh
```


### Prerequisites

* Go (Versión 1.20+, preferably 1.25)
* PostgreSQL engine.

### Environment variables (.env)

Project startup depends on `.env` variables.

| Var | Description | Type | Required | Domain |
| :--- | :--- | :--- | :--- | :--- |
| `DB_HOST` | Database host. | `string` | `true` | DB |
| `DB_NAME` | Database name. | `string` | `true` | DB |
| `DB_USER` | Database username. | `string` | `true` | DB |
| `DB_PASSWORD` | Database password. | `string` | `true` | DB |
| `DB_PORT` | Database port. | `int` | `true` | DB |
| `JWT_SECRET` | Secret string to authenticate/create tokens(Random). | `string` | `true` | Auth |
| `VAPIDPRIVATEKEY` | Push notification private key. | `string` | `true` | Notifications |
| `VAPIDPUBLICKEY` | Push notification private key. | `string` | `true` | Notifications |

***Note:*** `VAPID KEYS` can be generating using `/internal/domains/notifications/service.go:GenerateVAPIDKeys()` 

## API Endpoints

### 1. Authentication (`/auth`)

#### **Endpoint:**
Obtains a JWT token

*   **Method:** `POST`
*   **Endpoint:** `/auth`
*   **Middleware:** `ContentTypeAllowed("application/json")`
*   **Request body:**
    ```json
    {
        "application": "...", 
        "key": "...", 
        "password": "..."
    }
    ```
*   **Response (200 OK):** Returns auth token and expiration date
    ```json
    {
        "access_token": "...", 
        "expires_at": "...", 
        "expires_in": 900 
    }
    ```

### 2. User administration (`/users`)


#### **Endpoint: Store (Create user)**
Creates a new user and links it to an application, **Required** for sending notifications.

*   **Method:** `POST`
*   **Route:** `/users`
*   **Middleware:** JWT Middleware (`SecretJWT`) y Content Type JSON.
*   **Request body:**
    ```json
    {
        "user": "..."
    }
    ```
*   **Response (200 OK):** `{}` (Success)
*   **Errors:**
    *   `409 Conflict`: User already exists for the application.
    *   `401 Unauthorized`: Expired or invalid JWT Token.

#### **Endpoint: Remove**
Unlinks and removes a user from an application.

*   **Method:** `DELETE`
*   **Route:** `/users`
*   **Middleware:** JWT Middleware (`SecretJWT`) y Content Type JSON.
*   **Request body:**
    ```json
    {
        "user": "..."
    }
    ```
*   **Response (200 OK):** `{}` (Success)
*   **Errors:**
    *   `404 Not Found`: User not found.

### 3. Web Push Notifications (`/notifications`)

#### **Endpoint: Device subscribe**
Saves a devices subscription and links it to a user

*   **Method:** `POST`
*   **Route:** `/notifications/subscribe`
*   **Middleware:** JWT Middleware, Content Type JSON.
*   **Request body:**
    ```json
    {
        "user": "...", 
        "endpoint": "...",       // Provided by browser
        "p256dh": "...",         // Provided by browser
        "tag": "..."             // *Optional*, useful to send notifications when a user has multiple devices
    }
    ```
*   **Response (200 OK):** `{ "message": "Subscription saved" }`
*   **Errors:**
    *   `400 Bad Request`: Invalid or insufficient parameters.
    *   `409 Conflict`: The device is already subscribed.

#### **Endpoint: Send**
Sends a notification to one or more devices from a user, can filter by tag.

*   **Method:** `POST`
*   **Route:** `/notifications/send`
*   **Middleware:** JWT Middleware, Content Type JSON.
*   **Request body:**
    ```json
    {
        "user": "...", 
        "title": "...", 
        "message": "...", 
        "icon": "...url...", 
        "tag": "..." // *Optional*
    }
    ```
*   **Response (200 OK):** `{}`.
*   **Response partial (202 Accepted):**
    + Couldn't send notification to some devices and those devices subscriptions have been removed.
*   **Errors:**
    *   `401 Unauthorized`
    *   `400 Bad Request`: User data, tittle or message invalid/missing.
    *   `500 Internal Server Error`
