# PMA Remote Authentication System

## Overview

The PMA Remote Authentication System provides secure access control for the PMA Home Control system with intelligent connection-based authentication requirements. The system automatically detects the connection type and applies appropriate authentication rules.

## Key Features

### 🔐 Connection-Based Authentication
- **Localhost Access**: No authentication required for local development and direct access
- **Local Network Access**: No authentication required for devices on the same network
- **Remote Access**: Full user/password authentication required for external connections

### 👤 User Management
- **First-Time Setup**: Automatic user registration when no users exist
- **Admin Accounts**: Full user management with admin privileges
- **Secure Passwords**: Password strength validation and secure storage
- **Session Management**: JWT-based authentication with automatic token refresh

### 🏠 Local Hub Access
- **Automatic Bypass**: Localhost connections automatically authenticate as "Hub" user
- **Development Friendly**: No authentication barriers during local development
- **Production Ready**: Secure remote access with full authentication

## Architecture

### Backend Implementation

#### Authentication Middleware (`internal/api/middleware/remoteAuth.go`)
```go
// RemoteAuthMiddleware provides authentication that varies based on connection type
// - Localhost connections: No authentication required
// - Local network connections: No authentication required  
// - Remote connections: User/password authentication required
func RemoteAuthMiddleware(cfg *config.Config) gin.HandlerFunc
```

#### Authentication Handlers (`internal/api/handlers/auth.go`)
- `UserLogin`: Handle user/password authentication
- `UserRegister`: Create new user accounts
- `GetRemoteAuthStatus`: Check authentication requirements
- `GetUsers`: List all users (admin only)
- `DeleteUser`: Remove user accounts (admin only)

#### API Endpoints
```
POST /api/v1/auth/user/login          # User login
POST /api/v1/auth/user/register       # User registration
GET  /api/v1/auth/remote-status       # Check auth requirements
GET  /api/v1/users                    # List users (admin)
DELETE /api/v1/users/:id              # Delete user (admin)
```

### Frontend Implementation

#### Authentication Store (`src/stores/remoteAuthStore.ts`)
```typescript
interface RemoteAuthState {
  isAuthenticated: boolean
  user: User | null
  token: string | null
  requiresRemoteAuth: boolean
  connectionType: 'localhost' | 'local-network' | 'remote'
  authMode: 'local' | 'remote' | 'none'
}
```

#### Authentication Components
- `AuthRouter`: Intelligent routing to appropriate auth flow
- `RemoteLogin`: User/password login for remote access
- `UserRegistration`: First-time user creation
- `Login`: Legacy PIN-based authentication for local access

## Connection Detection

### Backend Logic
The system detects connection type based on client IP:

```go
// Check if it's localhost
if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
    connectionType = "localhost"
    requiresAuth = false
    isLocal = true
} else {
    // Check if it's local network (192.168.x.x, 10.x.x.x, 172.16-31.x.x)
    if strings.HasPrefix(clientIP, "192.168.") || 
       strings.HasPrefix(clientIP, "10.") || 
       (strings.HasPrefix(clientIP, "172.") && len(strings.Split(clientIP, ".")) == 4) {
        connectionType = "local-network"
        requiresAuth = false
        isLocal = true
    } else {
        connectionType = "remote"
        requiresAuth = true
        isLocal = false
    }
}
```

### Frontend Logic
The frontend checks authentication requirements on startup:

```typescript
const checkRemoteAuthRequired = async () => {
  const response = await apiService.checkRemoteAuthRequired()
  if (response.success && response.data) {
    const { requires_auth, connection_type } = response.data
    set({
      requiresRemoteAuth: requires_auth,
      connectionType: connection_type,
      authMode: requires_auth ? 'remote' : 'local'
    })
  }
}
```

## Authentication Flows

### 1. Localhost Access
```
User accesses via localhost → No authentication required → Direct access
```

### 2. Local Network Access
```
User accesses via local network → No authentication required → Direct access
```

### 3. Remote Access (No Users)
```
User accesses remotely → Check for users → No users exist → Show registration → Create admin → Login → Access granted
```

### 4. Remote Access (Users Exist)
```
User accesses remotely → Check for users → Users exist → Show login → Authenticate → Access granted
```

## Security Features

### Password Security
- **Minimum Length**: 8 characters
- **Complexity Requirements**: Uppercase, lowercase, number
- **Secure Storage**: bcrypt hashing
- **Strength Indicator**: Real-time password strength feedback

### Session Security
- **JWT Tokens**: Secure, time-limited authentication tokens
- **Automatic Refresh**: Tokens refresh automatically
- **Secure Transmission**: HTTPS-only communication
- **Session Timeout**: Configurable session expiration

### Access Control
- **Connection-Based**: Different rules for different connection types
- **Admin Privileges**: User management restricted to admins
- **Audit Logging**: Authentication events logged
- **Rate Limiting**: Protection against brute force attacks

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Sessions Table
```sql
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## Configuration

### Backend Configuration (`configs/config.yaml`)
```yaml
auth:
  enabled: true
  jwt_secret: "your-secret-key"
  session_duration: 3600  # 1 hour
  max_login_attempts: 5
  lockout_duration: 300   # 5 minutes
```

### Frontend Configuration
The frontend automatically detects authentication requirements and adapts accordingly.

## Usage Examples

### Testing the System

#### 1. Test Remote Auth Status
```bash
curl -X GET "http://localhost:3001/api/v1/auth/remote-status" \
  -H "Content-Type: application/json"
```

#### 2. Create First User
```bash
curl -X POST "http://localhost:3001/api/v1/auth/user/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!",
    "email": "admin@pma.local"
  }'
```

#### 3. User Login
```bash
curl -X POST "http://localhost:3001/api/v1/auth/user/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "Admin123!"
  }'
```

### Frontend Integration

#### 1. Check Authentication Requirements
```typescript
const { checkRemoteAuthRequired, requiresRemoteAuth } = useRemoteAuthStore()

useEffect(() => {
  checkRemoteAuthRequired()
}, [])
```

#### 2. Handle Authentication Flow
```typescript
const { userLogin, isAuthenticated } = useRemoteAuthStore()

const handleLogin = async (credentials) => {
  const success = await userLogin(credentials)
  if (success) {
    // Redirect to dashboard
  }
}
```

## Migration from Legacy System

### Backward Compatibility
- **PIN Authentication**: Still supported for local access
- **Legacy Endpoints**: Maintained for existing integrations
- **Gradual Migration**: Can be enabled/disabled per environment

### Migration Steps
1. **Enable Remote Auth**: Set `auth.enabled: true` in config
2. **Create Admin User**: Use registration endpoint or frontend
3. **Test Remote Access**: Verify authentication works from external networks
4. **Monitor Logs**: Check authentication events and errors
5. **Update Documentation**: Inform users of new authentication requirements

## Troubleshooting

### Common Issues

#### 1. Authentication Not Required for Remote Access
- Check `auth.enabled` in configuration
- Verify middleware is properly applied
- Check connection detection logic

#### 2. Frontend Shows Wrong Authentication Screen
- Clear browser cache and localStorage
- Check `checkRemoteAuthRequired()` response
- Verify API endpoints are accessible

#### 3. User Registration Fails
- Check database connectivity
- Verify password meets requirements
- Check for duplicate usernames

#### 4. Login Fails After Registration
- Verify user was created successfully
- Check password hashing
- Verify JWT token generation

### Debug Commands

#### Test Authentication System
```bash
./test_remote_auth_system.sh
```

#### Check Database Users
```bash
sqlite3 data/pma.db "SELECT id, username, created_at FROM users;"
```

#### Monitor Authentication Logs
```bash
tail -f logs/pma-backend.log | grep -i auth
```

## Security Considerations

### Best Practices
1. **Strong Passwords**: Enforce complex password requirements
2. **HTTPS Only**: Always use HTTPS in production
3. **Regular Updates**: Keep system and dependencies updated
4. **Monitor Access**: Log and monitor authentication events
5. **Backup Users**: Maintain backup admin accounts

### Security Features
- **bcrypt Hashing**: Secure password storage
- **JWT Tokens**: Time-limited authentication
- **Rate Limiting**: Protection against brute force
- **Connection Validation**: IP-based access control
- **Session Management**: Secure session handling

## Future Enhancements

### Planned Features
1. **Multi-Factor Authentication**: SMS/email verification
2. **OAuth Integration**: Google, GitHub, etc.
3. **Role-Based Access**: Different permission levels
4. **API Key Management**: For external integrations
5. **Audit Dashboard**: Authentication event monitoring

### Configuration Options
1. **Custom Network Ranges**: Configurable local network detection
2. **Session Policies**: Customizable session rules
3. **Password Policies**: Configurable password requirements
4. **Rate Limiting**: Adjustable rate limiting rules
5. **Logging Levels**: Configurable authentication logging

## Conclusion

The PMA Remote Authentication System provides a secure, flexible, and user-friendly authentication solution that adapts to different connection types while maintaining backward compatibility with existing systems. The implementation ensures security without compromising usability for local development and access. 