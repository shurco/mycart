-- +goose Up
-- +goose StatementBegin
-- The numeric merchantId and apiId SpectroCoin reports in every order callback.
--
-- SpectroCoin signs every merchant's callbacks with one global Merchant API
-- key, so a valid signature only proves that SpectroCoin signed the payload,
-- not that the payment reached this shop. These two fields are the only
-- identity fields in a callback that are both specific to one merchant account
-- and covered by the signature, so the callback handler compares them against
-- these settings before marking a cart paid. Without them a merchant could
-- settle this shop's carts with a callback for a payment made to their own
-- SpectroCoin account.
--
-- Empty on a fresh installation; spectrocoin_active is false there too, so no
-- callback is served until an operator fills them in. An installation that
-- already takes SpectroCoin payments will reject callbacks until its operator
-- copies both values across - they appear in every callback body and in the
-- project's dashboard.
INSERT INTO setting VALUES ('jk3nw34euqtn48c', 'spectrocoin_callback_merchant_id', '')
	ON CONFLICT (key) DO NOTHING;
INSERT INTO setting VALUES ('jhmu8qnmb7czohe', 'spectrocoin_callback_api_id', '')
	ON CONFLICT (key) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- By key, not by id: a future migration that reuses one of the ids above must
-- not lose its row here.
DELETE FROM setting WHERE key IN ('spectrocoin_callback_merchant_id', 'spectrocoin_callback_api_id');
-- +goose StatementEnd
