# checkout-api

[![Tests](https://github.com/easypmnt/checkout-api/actions/workflows/tests.yml/badge.svg)](https://github.com/easypmnt/checkout-api/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/easypmnt/checkout-api)](https://github.com/easypmnt/checkout-api/blob/main/LICENSE)

Payment API server based on the [Solana blockchain](https://solana.com).

## Features

- [x] Supports two payment flows: `classic` (via solana wallet adapter button) and `QR code`.
- [x] Webhooks for transaction status updates on the client's server.
- [x] SSE for transaction status updates on the client's browser.
- [x] Ability to use as a standalone API server or as a library.
- [x] Oauth2 authorization for client.
- [x] Support for authomated token swaps, if a customer pays with a token that the merchant does not support (using [Jupiter](https://jup.ag)).
- [x] A loyalty program for customers to earn bonuses for purchases and redeem them for discounts.
- [ ] Split payments between multiple merchants.
