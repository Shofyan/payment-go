# Postman Collection - Payment API

Comprehensive Postman collection for testing the high-throughput payment processing API.

## 📦 Files

- **Payment-API.postman_collection.json**: Complete API collection with all endpoints and tests
- **payment-api.postman_environment.json**: Local development environment variables
- **payment-api-production.postman_environment.json**: Production environment template

## 🚀 Quick Start

### 1. Import into Postman

1. Open Postman
2. Click **Import** button
3. Select all JSON files from this folder
4. Collections and Environments will be imported

### 2. Select Environment

1. Click environment dropdown (top right)
2. Select **Payment API - Local**
3. Verify `base_url` is set to `http://localhost`

### 3. Start the API

```bash
# From project root
docker-compose up -d
```

### 4. Run Collection

**Option A: Manual Testing**
- Expand folders and run individual requests
- View responses and test results

**Option B: Automated Testing (Runner)**
1. Click **Run Collection** button
2. Select **Payment API - High Throughput System**
3. Choose environment: **Payment API - Local**
4. Set iterations: 1 (for smoke test) or 100+ (for load test)
5. Click **Run**

## 📂 Collection Structure

### 1. Health & Status
- `GET /health` - Liveness probe
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### 2. Payments
- `POST /api/v1/payments` - Create payment (multiple methods)
  - Credit Card
  - Debit Card
  - Wallet
  - Bank Transfer
- `GET /api/v1/payments/{id}` - Get payment status

### 3. Error Scenarios
- Invalid amount (negative)
- Missing required fields
- Invalid payment ID format
- Payment not found

### 4. Load Testing
- **Rate Limit Test**: Trigger rate limiter (429 errors)
- **Backpressure Test**: Trigger system overload (503 errors)

### 5. Multiple Currencies
- Payments in USD, EUR, GBP, JPY

## 🧪 Testing Scenarios

### Smoke Test (Quick Validation)

Run the collection once to verify basic functionality:

```bash
# Using Postman Runner
Iterations: 1
Delay: 0ms
```

**Expected Results:**
- All health checks: ✅ Pass
- Payment creation: ✅ 202 Accepted
- Payment retrieval: ✅ 200 OK
- Error validation: ✅ 400/404

### Rate Limit Test

Test rate limiting (100 req/s per user):

```bash
# Using Postman Runner on "Rate Limit Test" request
Iterations: 300
Delay: 0ms
```

**Expected Behavior:**
- First ~100-200 requests: `202 Accepted`
- Remaining requests: `429 Too Many Requests`
- All 429 responses include `Retry-After` header

### Load Test (Backpressure)

Test system backpressure (worker pool queue full):

```bash
# Using Postman Runner on "Backpressure Test" request
Iterations: 2000
Delay: 0ms
```

**Expected Behavior:**
- Most requests: `202 Accepted`
- When queue full (>900 tasks): `503 Service Unavailable`
- System recovers as workers process tasks

## 🔧 Using Newman (CLI)

Install Newman:
```bash
npm install -g newman
```

### Run Entire Collection

```bash
newman run Payment-API.postman_collection.json \
  -e payment-api.postman_environment.json
```

### Run with Load Testing

```bash
# 1000 requests with no delay
newman run Payment-API.postman_collection.json \
  -e payment-api.postman_environment.json \
  -n 1000 \
  --delay-request 0
```

### Run Specific Folder

```bash
# Test only error scenarios
newman run Payment-API.postman_collection.json \
  -e payment-api.postman_environment.json \
  --folder "Error Scenarios"
```

### Generate HTML Report

```bash
# Install reporter
npm install -g newman-reporter-htmlextra

# Run with report
newman run Payment-API.postman_collection.json \
  -e payment-api.postman_environment.json \
  -r htmlextra \
  --reporter-htmlextra-export report.html
```

## 📊 Test Assertions

Each request includes automated test scripts:

### Example Test Scripts

```javascript
// Status code validation
pm.test("Status code is 202 Accepted", function () {
    pm.response.to.have.status(202);
});

// Response structure validation
pm.test("Response has payment_id", function () {
    var jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property('payment_id');
});

// Response time validation
pm.test("Response time is less than 500ms", function () {
    pm.expect(pm.response.responseTime).to.be.below(500);
});

// Save variables for subsequent requests
var jsonData = pm.response.json();
pm.environment.set("payment_id", jsonData.payment_id);
```

## 🎯 Load Testing with K6

For more advanced load testing, use K6:

```bash
# Install K6
brew install k6  # macOS
# or download from https://k6.io

# Create k6 script from Postman
postman-to-k6 Payment-API.postman_collection.json -e payment-api.postman_environment.json -o k6-script.js

# Run load test
k6 run k6-script.js
```

### K6 Load Test Example

```javascript
// k6-load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 50 },   // Ramp up to 50 users
    { duration: '1m', target: 100 },   // Stay at 100 users
    { duration: '30s', target: 200 },  // Peak at 200 users
    { duration: '1m', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests < 500ms
    http_req_failed: ['rate<0.05'],    // Error rate < 5%
  },
};

export default function () {
  const url = 'http://localhost/api/v1/payments';
  const payload = JSON.stringify({
    user_id: `user_${__VU}_${__ITER}`,
    merchant_id: 'merchant_load_test',
    amount: 10000,
    currency: 'USD',
    method: 'CREDIT_CARD',
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(url, payload, params);

  check(res, {
    'status is 202 or 429 or 503': (r) => [202, 429, 503].includes(r.status),
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  sleep(0.1);
}
```

Run:
```bash
k6 run k6-load-test.js
```

## 🔍 Monitoring During Tests

### View Logs

```bash
# API logs
docker-compose logs -f api-1 api-2 api-3

# NGINX logs
docker-compose logs -f nginx
```

### View Metrics

- **Prometheus**: http://localhost:9090
  - Query: `rate(http_requests_total[1m])`
  - Query: `worker_pool_queue_size`
  - Query: `rate_limiter_requests_denied_total`

- **Grafana**: http://localhost:3000
  - Login: admin/admin
  - Create dashboard with above metrics

## 🚨 Expected Test Results

### Successful Scenarios

| Test | Expected Status | Expected Time |
|------|----------------|---------------|
| Health Check | 200 OK | < 50ms |
| Create Payment | 202 Accepted | < 200ms |
| Get Payment | 200 OK | < 100ms |

### Error Scenarios

| Test | Expected Status | Reason |
|------|----------------|--------|
| Negative Amount | 400 Bad Request | Validation failed |
| Missing Field | 400 Bad Request | Required field |
| Invalid UUID | 400 Bad Request | Format error |
| Not Found | 404 Not Found | Payment doesn't exist |

### Load Testing

| Test | Expected Behavior |
|------|-------------------|
| Rate Limit | First ~100-200 requests succeed, then 429 |
| Backpressure | 503 when queue > 90% full (>900 tasks) |
| Recovery | System accepts requests after queue drains |

## 📝 Environment Variables

### Local Environment

```json
{
  "base_url": "http://localhost",
  "payment_id": "",  // Auto-populated after payment creation
  "user_id": ""      // Auto-generated per request
}
```

### Production Environment

```json
{
  "base_url": "https://api.payment.example.com",
  "api_key": "your-secret-api-key",
  "payment_id": "",
  "user_id": ""
}
```

## 🛠️ Troubleshooting

### Issue: Connection Refused

**Solution**: Ensure Docker services are running
```bash
docker-compose ps
docker-compose up -d
```

### Issue: All Requests Return 503

**Solution**: System is overloaded, reduce concurrency
```bash
# Check worker pool status
curl http://localhost/metrics | grep worker_pool_queue_size

# Restart services
docker-compose restart
```

### Issue: Rate Limit Not Working

**Solution**: Requests must use same user_id
- Check pre-request script sets `user_id` correctly
- For rate limit test, use fixed user_id

### Issue: Tests Failing

**Solution**: Check environment selection
1. Verify correct environment is selected
2. Check `base_url` variable
3. Ensure API is accessible

## 📚 Additional Resources

- [Postman Documentation](https://learning.postman.com/docs/)
- [Newman CLI](https://learning.postman.com/docs/running-collections/using-newman-cli/)
- [K6 Load Testing](https://k6.io/docs/)
- [Project Architecture](../ARCHITECTURE.md)
- [API README](../README.md)

## 🤝 Contributing

To add new test scenarios:

1. Create new request in appropriate folder
2. Add test scripts for validation
3. Add description explaining the test
4. Update this README with new test case
5. Test with Newman to ensure CLI compatibility

---

**Happy Testing! 🚀**
