# checkout-api

[![Go Reference](https://pkg.go.dev/badge/github.com/easypmnt/checkout-api.svg)](https://pkg.go.dev/github.com/easypmnt/checkout-api)
[![Go Report Card](https://goreportcard.com/badge/github.com/easypmnt/checkout-api)](https://goreportcard.com/report/github.com/easypmnt/checkout-api)
[![Tests](https://github.com/easypmnt/checkout-api/actions/workflows/tests.yml/badge.svg)](https://github.com/easypmnt/checkout-api/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/easypmnt/checkout-api)](https://github.com/easypmnt/checkout-api/blob/main/LICENSE)

Payment API server based on the [Solana blockchain](https://solana.com).

### Demo

[Checkout page](https://example-checkout.easypmnt.com) is connected to the mainnet node and demonstrates purchasing via QR code, accruing and applying bonuses, and automatic token swapping if needed.

[![Demo](./example.gif)](https://example-checkout.easypmnt.com)


## Features

- [x] Supports two payment flows: `classic` (via solana wallet adapter button) and `QR code`.
- [x] Webhooks for transaction status updates on the client's server.
- [x] Transaction status updates via websocket (useful for client-side widgets).
- [x] Ability to use as a standalone API server or as a library.
- [x] Oauth2 authorization for client.
- [x] Support for authomated token swaps, if a customer pays with a token that the merchant does not support (using [Jupiter](https://jup.ag)).
- [x] A loyalty program for customers to earn bonuses for purchases and redeem them for discounts.

### Comming soon

- [ ] Project documentation, in addition to the default on [pkg.go.dev](https://pkg.go.dev/github.com/easypmnt/checkout-api)
- [ ] Split payments between multiple merchants.
- [ ] Typescript/Javascript SDK and widget for quick integration into a project.
- [ ] Plugins for popular CMS (e.g., WordPress, PrestaShop, etc).
- [ ] Web UI to configure payment server options.
- [ ] More options for the loyalty program: bonus cards with different discount levels or additional benefits, bonus for N purchases, etc.


## How it works

[![How it works](./how_it_works.png)](https://docs.solanapay.com/core/transfer-request/merchant-integration)
According to the [Solanapay protocol](https://docs.solanapay.com/core/transfer-request/merchant-integration).
