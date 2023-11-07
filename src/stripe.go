package src

import (
	"errors"
	"purity-vision-filter/src/config"
	"purity-vision-filter/src/license"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/usagerecord"
)

func IncrementSubscriptionMeter(lic *license.License, quantity int64) error {
	if config.StripeKey == "" {
		return errors.New("STRIPE_KEY env var not found")
	}

	stripe.Key = config.StripeKey

	s, err := fetchStripeSubscription(lic)
	if err != nil {
		return err
	}

	params := &stripe.UsageRecordParams{
		SubscriptionItem: &s.Items.Data[0].ID,
		Action:           stripe.String(string(stripe.UsageRecordActionIncrement)),
		Quantity:         &quantity,
	}

	_, err = usagerecord.New(params)

	return err
}

func fetchStripeSubscription(lic *license.License) (*stripe.Subscription, error) {
	if config.StripeKey == "" {
		return nil, errors.New("STRIPE_KEY env var not found")
	}

	stripe.Key = config.StripeKey

	sub, err := subscription.Get(lic.SubscriptionID, nil)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
