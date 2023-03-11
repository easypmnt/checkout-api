# checkout-api

[![Tests](https://github.com/easypmnt/checkout-api/actions/workflows/tests.yml/badge.svg)](https://github.com/easypmnt/checkout-api/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/easypmnt/checkout-api)](https://github.com/easypmnt/checkout-api/blob/main/LICENSE)

Payment API server based on the [Solana blockchain](https://solana.com).

## Features

- [x] Create checkout transaction with a single API call.
- [x] Track transaction status updates.
- [x] Webhooks for transaction status updates.
- [x] Ability to use as standalone API server or as a library.
- [x] Client credentials oauth2 authorization flow.
- [x] Support for authomated token swaps, if customer pays with a token that is not supported by the merchant (using [Jupiter](https://jup.ag)).
- [x] Loyal program for customers to earn bonuses for purchases and redeem them for discounts.
