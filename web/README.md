# Web Interface

This directory contains the web interface for the Payment API, built with HTMX for dynamic interactions.

## Directory Structure

```
web/
├── templates/          # HTML templates
│   ├── index.html              # Homepage
│   ├── create-payment.html     # Create payment page
│   ├── get-payment.html        # Get payment page
│   ├── payment-result.html     # Payment creation result (HTMX partial)
│   ├── payment-details.html    # Payment details (HTMX partial)
│   ├── payment-page.html       # Full payment details page
│   └── error.html              # Error template (HTMX partial)
└── static/            # Static assets
    └── css/
        └── style.css          # Styles for the web interface
```

## Features

### 🎨 Modern UI
- Clean, responsive design
- Mobile-friendly layout
- Professional styling with CSS variables
- Smooth transitions and animations

### ⚡ HTMX Integration
- Dynamic form submissions without page reloads
- Real-time status updates
- Partial page updates for better UX
- Loading indicators during requests

### 📱 Pages

#### 1. Home Page (`/` or `/web/`)
- Overview of the Payment API
- Feature highlights
- API endpoint documentation
- Quick action buttons
- Real-time system status (auto-refreshes every 30s)

#### 2. Create Payment (`/web/create`)
- Form to create new payments
- Fields:
  - User ID
  - Merchant ID
  - Amount (in cents)
  - Currency (USD, EUR, GBP, JPY, CAD, AUD)
  - Payment Method (credit_card, debit_card, bank_transfer, paypal, cryptocurrency)
- Real-time validation
- HTMX-powered submission
- Instant result display with payment ID
- Copy payment ID to clipboard

#### 3. Get Payment (`/web/get`)
- Search form for payment lookup
- UUID format validation
- HTMX-powered search
- Detailed payment information display
- Refresh status button
- Status-specific icons and messages

#### 4. Payment Details Page (`/web/payments/{id}`)
- Direct URL access to payment details
- Full payment information:
  - Payment ID
  - User ID
  - Merchant ID
  - Amount and Currency
  - Payment Method
  - Status (with color-coded badges)
  - Transaction ID (if available)
  - Created At / Updated At timestamps
- Refresh button for status updates

## Status Indicators

The interface uses color-coded status badges:

- 🟡 **Pending**: Payment awaiting processing
- 🔵 **Processing**: Payment currently being processed
- 🟢 **Completed**: Payment successfully completed
- 🔴 **Failed**: Payment failed

## HTMX Attributes Used

- `hx-post`: POST requests for form submissions
- `hx-get`: GET requests for fetching data
- `hx-target`: Specify where to insert response
- `hx-swap`: Control how content is swapped
- `hx-indicator`: Show loading spinner
- `hx-trigger`: Control when requests are made
- `hx-vals`: Pass additional values with requests

## CSS Features

- CSS custom properties (variables) for theming
- Responsive grid layouts
- Flexbox for alignment
- Smooth transitions
- Box shadows for depth
- Mobile-first approach
- Loading animations

## Browser Requirements

- Modern browsers with ES6+ support
- JavaScript enabled (for HTMX)
- CSS Grid and Flexbox support

## Development

### Adding New Pages

1. Create HTML template in `web/templates/`
2. Add handler method in `internal/interface/http/handler/web_handler.go`
3. Register route in `internal/interface/http/router/router.go`

### Modifying Styles

Edit `web/static/css/style.css` to modify:
- Colors (CSS variables in `:root`)
- Layout and spacing
- Component styles
- Responsive breakpoints

## Testing

Access the web interface:
- Homepage: http://localhost:8080/
- Create Payment: http://localhost:8080/web/create
- Get Payment: http://localhost:8080/web/get

## API Integration

The web interface communicates with these backend endpoints:
- `POST /web/payments/create` - Create payment via form
- `POST /web/payments/get` - Get payment via form
- `GET /web/payments/{id}` - Get payment details page
- `GET /health` - Health check (auto-refresh on homepage)

## Security Considerations

- Form validation on both client and server side
- CSRF protection (handled by middleware)
- Rate limiting applied to all requests
- Input sanitization in templates
- No sensitive data in URLs (except payment IDs)

## Performance

- Minimal JavaScript (only HTMX ~14KB gzipped)
- Efficient partial page updates
- CSS-based animations (hardware accelerated)
- Optimized for fast page loads

## Future Enhancements

Potential improvements:
- [ ] Payment history/list view
- [ ] Advanced search filters
- [ ] Bulk operations
- [ ] Real-time webhooks display
- [ ] Dashboard with analytics
- [ ] Dark mode toggle
- [ ] Export payment data
- [ ] Multi-language support
