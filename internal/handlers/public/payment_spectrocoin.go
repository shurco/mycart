package handlers

import (
	"errors"
	"strconv"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
)

// spectrocoinVerifier checks the RSA signature of a SpectroCoin order callback.
// It is a parameter of paymentCallback rather than a direct call so that the
// callback flow stays testable: the real verifier checks the payload against
// SpectroCoin's global public key, whose private half nobody here holds, so no
// test can produce a payload it would accept.
type spectrocoinVerifier func(*litepay.CallbackSpectrocoin) error

// spectrocoinPayment turns a signature-verified SpectroCoin order callback into
// the payment it describes for cartID, or refuses it.
//
// The signature alone is not enough to trust the payload. SpectroCoin signs
// every merchant's callbacks with one global Merchant API key, so a merchant can
// pay their own account, take the callback SpectroCoin sends them and post it
// here, choosing orderId, receiveCurrency and receiveAmount to match a cart of
// ours: a valid signature proves SpectroCoin signed the payload, not that the
// payment reached this shop. Of the identity fields a callback carries only
// merchantId and apiId are both specific to one merchant account and covered by
// the signature - userId and merchantApiId sit outside it, so the caller may set
// them to anything and comparing those would be no obstacle. Binding the signed
// pair to the configured one is what stops a foreign merchant settling this
// shop's cart.
//
// Every error it returns is a static message, safe to hand back to the
// unauthenticated caller; the values that would reveal the shop's configured
// identity go to the log instead.
func spectrocoinPayment(log *logging.Log, cb *litepay.CallbackSpectrocoin, setting *models.Spectrocoin, cartID string) (*litepay.Payment, error) {
	// An unset identity is refused rather than compared: a callback claiming
	// merchantId=0 would otherwise be an exact match for a shop that has not
	// filled the values in yet, which is the same hole wearing a different hat.
	if setting.CallbackMerchantID <= 0 || setting.CallbackApiID <= 0 {
		log.Error().Msgf("spectrocoin callback for cart %s refused: the callback merchant identity is not configured", cartID)
		return nil, errors.New("callback merchant identity is not configured")
	}
	if cb.MerchantID != setting.CallbackMerchantID || cb.ApiID != setting.CallbackApiID {
		log.Error().Msgf("spectrocoin callback merchant mismatch for cart %s: got merchantId=%d apiId=%d, want merchantId=%d apiId=%d",
			cartID, cb.MerchantID, cb.ApiID, setting.CallbackMerchantID, setting.CallbackApiID)
		return nil, errors.New("callback merchant does not match configuration")
	}
	if cb.OrderID != cartID {
		log.Error().Msgf("spectrocoin callback order does not match cart %s", cartID)
		return nil, errors.New("callback order does not match cart")
	}

	return &litepay.Payment{
		CartID:        cartID,
		PaymentSystem: litepay.SPECTROCOIN,
		Status:        litepay.StatusPayment(litepay.SPECTROCOIN, strconv.Itoa(cb.Status)),
		// The callback carries no order UUID, so the signed order request id is
		// the per-order reference available. The unsigned merchantApiId stored
		// here before was the caller's to choose.
		MerchantID: strconv.Itoa(cb.OrderRequestID),
		Coin: &litepay.Coin{
			AmountTotal: cb.ReceiveAmount,
			Currency:    cb.ReceiveCurrency,
		},
	}, nil
}
