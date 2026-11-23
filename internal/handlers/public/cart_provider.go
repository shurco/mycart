package handlers

import (
	"context"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/litepay"
)

// providerSession builds the provider session the shop has configured for
// system. active is false when the operator has switched that provider off;
// the session is nil for a system that has none (an unknown one).
func providerSession(ctx context.Context, db *queries.Base, pay litepay.Cfg, system litepay.PaymentSystem) (session litepay.LitePay, active bool, err error) {
	switch system {
	case litepay.STRIPE:
		setting, err := queries.GetSettingByGroup[models.Stripe](ctx, db)
		if err != nil || !setting.Active {
			return nil, false, err
		}
		return pay.Stripe(setting.SecretKey), true, nil

	case litepay.PAYPAL:
		setting, err := queries.GetSettingByGroup[models.Paypal](ctx, db)
		if err != nil || !setting.Active {
			return nil, false, err
		}
		return pay.Paypal(setting.ClientID, setting.SecretKey), true, nil

	case litepay.SPECTROCOIN:
		setting, err := queries.GetSettingByGroup[models.Spectrocoin](ctx, db)
		if err != nil || !setting.Active {
			return nil, false, err
		}
		return pay.Spectrocoin(setting.MerchantID, setting.ProjectID, setting.PrivateKey), true, nil

	case litepay.COINBASE:
		setting, err := queries.GetSettingByGroup[models.Coinbase](ctx, db)
		if err != nil || !setting.Active {
			return nil, false, err
		}
		return pay.Coinbase(setting.ApiKey), true, nil

	case litepay.DUMMY:
		// Dummy provider is always active and only for free carts (already validated above)
		return pay.Dummy(), true, nil
	}
	return nil, true, nil
}
