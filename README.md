# NetBox DNS Update Service

This service listens for webhooks from NetBox (specifically the NetBox DNS plugin) and updates DNS records accordingly using `nsupdate`. It supports creating, updating, and deleting both forward DNS records (e.g., A, AAAA) and associated PTR records.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Building the Docker Image](#building-the-docker-image)
- [Running the Docker Container](#running-the-docker-container)
- [Environment Variables](#environment-variables)
- [Logging](#logging)
- [Endpoints](#endpoints)
- [License](#license)

## Prerequisites

- **Go 1.20** or later (if you plan to build the application manually)
- **Docker** (if you plan to build and run using Docker)
- **NetBox** with the NetBox DNS plugin configured to send webhooks
- **TSIG Key** for secure DNS updates

## Configuration

The application is configured via environment variables. The main configuration parameters are:

- `BIND_SERVER_ADDRESS`: The address of the DNS server to update (e.g., `127.0.0.1:53`).
- `TSIG_KEY_FILE`: Path to the TSIG key file inside the container (e.g., `/app/tsig.key`).
- `WEBHOOK_LISTEN_ADDRESS`: The address and port the application listens on for webhooks (e.g., `:8080`).
- `LOG_LEVEL`: The logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- `LOG_FORMAT`: The logging format (`logfmt`, `json`).

## Building the Docker Image

1. **Clone the repository** (if you haven't already):

   ```bash
   git clone https://github.com/yourusername/netbox-dnsupdate.git
   cd netbox-dnsupdate
   ```

2. **Place your TSIG key file** in the project directory:

   - Copy your TSIG key file to the project directory and name it `tsig.key`.
   - **Note**: Ensure that your TSIG key file is secure and not committed to version control.

3. **Build the Docker image**:

   ```bash
   docker build -t netbox-dnsupdate:latest .
   ```

## Running the Docker Container

Run the container with the necessary environment variables and port mappings. Here's an example:

```bash
docker run -d \
  --name netbox-dnsupdate \
  -p 8080:8080 \
  -e BIND_SERVER_ADDRESS="dns-server.example.com:53" \
  -e TSIG_KEY_FILE="/app/tsig.key" \
  -e WEBHOOK_LISTEN_ADDRESS=":8080" \
  -e LOG_LEVEL="INFO" \
  -e LOG_FORMAT="json" \
  netbox-dnsupdate:latest
```

**Explanation of the options:**

- `-d`: Run the container in detached mode.
- `--name netbox-dnsupdate`: Name the container for easier management.
- `-p 8080:8080`: Map port 8080 of the container to port 8080 on the host.
- `-e`: Set environment variables to configure the application.

## Environment Variables

- `BIND_SERVER_ADDRESS`: Address and port of the DNS server (default: `127.0.0.1:53`).
- `TSIG_KEY_FILE`: Path to the TSIG key file inside the container (required).
- `WEBHOOK_LISTEN_ADDRESS`: Address and port for the webhook listener (default: `:8080`).
- `LOG_LEVEL`: Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`; default: `INFO`).
- `LOG_FORMAT`: Logging format (`logfmt`, `json`; default: `logfmt`).

## Logging

The application supports structured logging in both `logfmt` and `json` formats. Configure the logging format using the `LOG_FORMAT` environment variable.

- **Log Levels**: Control the verbosity of logs using the `LOG_LEVEL` variable.
- **Sample Log Entry (logfmt)**:

  ```
  ts=2024-11-06T11:36:13Z level=info msg="Processed DNS record" event=created fqdn=test1.example.com. record_type=A value=10.5.199.71 ttl=300 user=jdoe request_id=a767bb9b record_id=311
  ```

- **Sample Log Entry (JSON)**:

  ```json
  {
    "ts": "2024-11-06T11:36:13Z",
    "level": "info",
    "msg": "Processed DNS record",
    "event": "created",
    "fqdn": "test1.example.com.",
    "record_type": "A",
    "value": "10.5.199.71",
    "ttl": 300,
    "user": "jdoe",
    "request_id": "a767bb9b",
    "record_id": 311
  }
  ```

## Endpoints

- `/webhook`: The main endpoint that receives webhook POST requests from NetBox.
- `/healthz`: Health check endpoint that responds with `OK`.
- `/ready`: Readiness check endpoint that responds with `READY`.

## Health Checks

You can use the `/healthz` and `/ready` endpoints to monitor the application's health and readiness. For example:

```bash
curl http://localhost:8080/healthz
# Output: OK

curl http://localhost:8080/ready
# Output: READY
```

## Configuring NetBox Webhooks

To have NetBox send webhooks to this service:

1. **Create a Webhook** in NetBox:

   - Go to **Admin > NetBox DNS > Webhooks**.
   - Click **Add a new webhook**.
   - Configure the webhook with the following settings:
     - **Name**: Descriptive name (e.g., `DNS Update Webhook`).
     - **Content types**: Select the DNS record types you want to send (e.g., `Record`).
     - **Type of events**: Choose `Created`, `Updated`, `Deleted`.
     - **URL**: The URL of the webhook endpoint (e.g., `http://netbox-dnsupdate:8080/webhook`).
     - **HTTP method**: `POST`.
     - **Additional HTTP headers**: Add any required headers.
     - **SSL verification**: Enable or disable based on your setup.

2. **Test the Webhook**:

   - Create, update, or delete a DNS record in NetBox.
   - Verify that the application logs show the processing of the event.
   - Ensure that the DNS server has been updated accordingly.

## Security Considerations

- **TSIG Key Management**: Ensure that your TSIG key is kept secure. Do not commit it to version control.
- **Network Security**: Secure the communication between NetBox and this service, and between this service and the DNS server.
- **Authentication**: Implement authentication for the webhook endpoint if exposed over public networks.

## Troubleshooting

- **Logs**: Check the application logs for any errors or warnings.
- **DNS Updates**: Ensure that the DNS server allows updates from the application and that the TSIG key is correct.
- **Network Connectivity**: Verify that the application can reach the DNS server and that NetBox can reach the webhook endpoint.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
