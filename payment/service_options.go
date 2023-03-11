package payment

import "strings"

// WithSolanaPayBaseURI sets the base URI to use in QR code payments.
func WithSolanaPayBaseURI(baseURI string) ServiceOption {
	return func(s *Service) {
		if baseURI == "" {
			panic("base URI can't be empty")
		}
		if !strings.HasPrefix(baseURI, "https://") {
			panic("base URI should start with https://")
		}

		s.solanaPayBaseURI = strings.TrimRight(strings.TrimSpace(baseURI), "/")
	}
}

// WithDefaultMerchantSettings sets the default merchant settings.
func WithDefaultMerchantSettings(base58Addr string, applyBonus bool, maxBonus uint64, maxBonusPerc uint16, mint string) ServiceOption {
	return func(s *Service) {
		s.defaultMerchantSettings = MerchantSettings{
			WalletAddress: base58Addr,
			ApplyBonus:    applyBonus,
			MaxBonus:      maxBonus,
			MaxBonusPerc:  maxBonusPerc,
			BonusMintAddr: mint,
		}
	}
}

// WithDefaultMerchantWalletAddress sets the default merchant wallet address.
func WithDefaultMerchantWalletAddress(base58Addr string) ServiceOption {
	return func(s *Service) {
		s.defaultMerchantSettings.WalletAddress = base58Addr
	}
}

// WithDefaultMerchantApplyBonus sets the default merchant apply bonus.
func WithDefaultMerchantApplyBonus(applyBonus bool) ServiceOption {
	return func(s *Service) {
		s.defaultMerchantSettings.ApplyBonus = applyBonus
	}
}

// WithDefaultMerchantMaxBonus sets the default merchant max bonus.
func WithDefaultMerchantMaxBonus(maxBonus uint64) ServiceOption {
	return func(s *Service) {
		s.defaultMerchantSettings.MaxBonus = maxBonus
	}
}

// WithDefaultMerchantMaxBonusPerc sets the default merchant max bonus percentage.
func WithDefaultMerchantMaxBonusPerc(maxBonusPerc uint16) ServiceOption {
	return func(s *Service) {
		s.defaultMerchantSettings.MaxBonusPerc = maxBonusPerc
	}
}

// WithDefaultMerchantBonusMintAddr sets the default merchant bonus mint address.
func WithDefaultMerchantBonusMintAddr(mint string) ServiceOption {
	return func(s *Service) {
		s.defaultMerchantSettings.BonusMintAddr = mint
	}
}