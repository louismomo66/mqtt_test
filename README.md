# MQTT Backend with PostgreSQL and Mosquitto

This project provides a Go-based MQTT backend service that stores device data in PostgreSQL and uses Mosquitto as the MQTT broker.

## Prerequisites

- Docker and Docker Compose installed on your system
- Go 1.24+ (for local development)

## Quick Start with Docker

1. **Clone the repository and navigate to the project directory**

2. **Start all services using Docker Compose:**
   ```bash
   docker-compose up -d
   ```

   This will start:
   - PostgreSQL database on port 5432
   - Mosquitto MQTT broker on port 1883 (MQTT) and 9001 (WebSocket)
   - Go MQTT backend application on port 8080

3. **Check service status:**
   ```bash
   docker-compose ps
   ```

4. **View logs:**
   ```bash
   # View all logs
   docker-compose logs -f
   
   # View specific service logs
   docker-compose logs -f app
   docker-compose logs -f postgres
   docker-compose logs -f mosquitto
   ```

## Configuration

### Environment Variables

The application uses the following environment variables:

- `POSTGRES_DSN`: PostgreSQL connection string (default: `postgresql://mqtt_user:mqtt_password@localhost:5432/mqtt_db?sslmode=disable`)
- `MQTT_BROKER`: MQTT broker URL (default: `tcp://localhost:1883`)

### Database Configuration

- **Database**: `mqtt_db`
- **Username**: `mqtt_user`
- **Password**: `mqtt_password`
- **Port**: 5432

### MQTT Configuration

- **Broker**: Mosquitto
- **Port**: 1883 (MQTT), 9001 (WebSocket)
- **Authentication**: Anonymous (for development)

## Development

### Local Development

1. **Start only the infrastructure services:**
   ```bash
   docker-compose up -d postgres mosquitto
   ```

2. **Run the Go application locally:**
   ```bash
   go run cmd/api/main.go
   ```

### Building the Application

```bash
# Build the Docker image
docker-compose build app

# Or build locally
go build -o mqtt-backend ./cmd/api
```

## MQTT Topics

The application subscribes to the following MQTT topics:

- `device/logs/+/data` - Standard device data messages
- `device/logs/+/chunked/#` - Chunked device data messages

## Database Schema

The application automatically creates the following tables:

- `devices` - Device information
- `device_data` - Device sensor data and logs

## Stopping Services

```bash
# Stop all services
docker-compose down

# Stop and remove volumes (WARNING: This will delete all data)
docker-compose down -v
```

## Troubleshooting

### Database Connection Issues

1. Check if PostgreSQL is running:
   ```bash
   docker-compose ps postgres
   ```

2. Check PostgreSQL logs:
   ```bash
   docker-compose logs postgres
   ```

3. Test database connection:
   ```bash
   docker-compose exec postgres psql -U mqtt_user -d mqtt_db
   ```

### MQTT Connection Issues

1. Check if Mosquitto is running:
   ```bash
   docker-compose ps mosquitto
   ```

2. Check Mosquitto logs:
   ```bash
   docker-compose logs mosquitto
   ```

3. Test MQTT connection:
   ```bash
   docker-compose exec mosquitto mosquitto_pub -h localhost -t test -m "hello"
   ```

### Application Issues

1. Check application logs:
   ```bash
   docker-compose logs app
   ```

2. Restart the application:
   ```bash
   docker-compose restart app
   ```

## Production Considerations

For production deployment, consider:

1. **Security**: Enable MQTT authentication and TLS
2. **Database**: Use managed PostgreSQL service
3. **Monitoring**: Add health checks and monitoring
4. **Backup**: Implement database backup strategy
5. **Scaling**: Use load balancers and multiple instances
