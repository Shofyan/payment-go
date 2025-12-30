# Grafana Dashboard for Payment API

This directory contains the Grafana dashboard configuration and provisioning files for monitoring the Payment API application.

## Overview

The Grafana dashboard provides comprehensive monitoring of:
- **HTTP Metrics**: Request rates, response times, status codes
- **Payment Metrics**: Transaction counts and rates by status
- **Infrastructure Metrics**: Circuit breaker, rate limiter, worker pools
- **System Metrics**: Memory usage, goroutines, database connections
- **Instance Monitoring**: Performance across all 3 API instances

## Quick Start

### 1. Start the Services

```bash
docker-compose up -d
```

### 2. Access Grafana

Open your browser and navigate to:
```
http://localhost:3000
```

**Default Credentials:**
- Username: `admin`
- Password: `admin`

### 3. View the Dashboard

The dashboard will be automatically provisioned and available at:
```
Home → Dashboards → Payment API Dashboard
```

## Dashboard Panels

### Top Row - Key Metrics
1. **Total Request Rate**: Overall requests per second across all instances
2. **Success Rate (2xx)**: Percentage of successful requests
3. **P95 Response Time**: 95th percentile response time in milliseconds
4. **Error Rate (5xx)**: Server error rate

### HTTP Monitoring
5. **Request Rate by Endpoint**: Traffic breakdown by API endpoints
6. **Response Time Percentiles**: P50, P95, P99 latencies by instance
7. **HTTP Status Codes**: Distribution of response codes over time
8. **Request Rate by Instance**: Load distribution across instances

### Business Metrics
9. **Payment Transactions by Status**: Cumulative payment counts (pending, completed, failed)
10. **Payment Transaction Rate**: Real-time payment processing rate

### Infrastructure Monitoring
11. **Circuit Breaker Activity**: State changes indicating service failures
12. **Rate Limiter Activity**: Allowed vs blocked requests
13. **Worker Pool Status**: Active workers and queue size
14. **Database Connection Pool**: Active and idle connections

### System Resources
15. **Memory Usage by Instance**: Memory allocation per instance
16. **Goroutines by Instance**: Number of concurrent goroutines

## Data Source

The dashboard uses Prometheus as the data source, which is automatically configured via:
- **File**: `grafana/provisioning/datasources/prometheus.yml`
- **URL**: `http://prometheus:9090`
- **Scrape Interval**: 15 seconds

## Customization

### Editing the Dashboard

1. Open the dashboard in Grafana
2. Click the gear icon (⚙️) in the top right → "Dashboard settings"
3. Make your changes
4. Click "Save dashboard"

To persist changes:
1. Click "Share" → "Export" → "Save to file"
2. Replace `grafana/provisioning/dashboards/payment-api-dashboard.json` with the exported file

### Adding Custom Panels

The dashboard supports any metrics exposed by your application. Common metric types:

**Counters:**
```promql
rate(metric_name_total[5m])
```

**Gauges:**
```promql
metric_name
```

**Histograms:**
```promql
histogram_quantile(0.95, sum(rate(metric_name_bucket[5m])) by (le))
```

## Metrics Reference

### HTTP Metrics
- `http_requests_total`: Total HTTP requests (labels: method, path, status, instance)
- `http_request_duration_seconds`: Request duration histogram

### Payment Metrics
- `payment_total`: Total payment transactions (labels: status)

### Infrastructure Metrics
- `circuitbreaker_state_changes_total`: Circuit breaker state changes
- `ratelimiter_requests_blocked_total`: Blocked requests by rate limiter
- `ratelimiter_requests_allowed_total`: Allowed requests by rate limiter
- `workerpool_active_workers`: Current active workers
- `workerpool_queue_size`: Current queue size

### Database Metrics
- `database_connections_active`: Active database connections
- `database_connections_idle`: Idle database connections

### Go Runtime Metrics
- `go_memstats_alloc_bytes`: Memory allocated
- `go_goroutines`: Number of goroutines

## Alerts (Optional)

You can configure alerts in Grafana:

1. Open a panel
2. Click the panel title → "Edit"
3. Go to "Alert" tab
4. Configure alert conditions
5. Set notification channels (Slack, email, etc.)

Example alert conditions:
- Error rate > 1%
- P95 response time > 500ms
- Memory usage > 80%
- Circuit breaker state changes

## Troubleshooting

### Dashboard Not Loading

1. Check if Grafana is running:
   ```bash
   docker-compose ps grafana
   ```

2. Check Grafana logs:
   ```bash
   docker-compose logs grafana
   ```

3. Verify Prometheus is accessible:
   ```bash
   curl http://localhost:9090/-/healthy
   ```

### No Data in Panels

1. Check Prometheus targets:
   - Open http://localhost:9090/targets
   - Ensure all `payment-api` targets are "UP"

2. Verify metrics are being scraped:
   - Open http://localhost:9090/graph
   - Query: `http_requests_total`

3. Check API instances are exposing metrics:
   ```bash
   curl http://localhost:80/metrics
   ```

### Dashboard Changes Not Persisting

The dashboard is provisioned from the JSON file. To make permanent changes:
1. Export the modified dashboard from Grafana
2. Save it to `grafana/provisioning/dashboards/payment-api-dashboard.json`
3. Restart Grafana: `docker-compose restart grafana`

## Performance Tips

1. **Adjust Time Range**: Use shorter time ranges (15m, 30m) for better performance
2. **Increase Refresh Interval**: Change from 10s to 30s or 1m in dashboard settings
3. **Limit Cardinality**: Avoid high-cardinality labels in metrics
4. **Use Recording Rules**: Pre-compute complex queries in Prometheus

## Additional Resources

- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Query Language (PromQL)](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Best Practices for Metrics](https://prometheus.io/docs/practices/naming/)
