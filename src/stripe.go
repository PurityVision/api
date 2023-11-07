package src

import (
	"errors"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/usagerecord"
)

func IncrementSubscriptionMeter(lic *License, quantity int64) error {
	if StripeKey == "" {
		return errors.New("STRIPE_KEY env var not found")
	}

	stripe.Key = StripeKey

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

func fetchStripeSubscription(lic *License) (*stripe.Subscription, error) {
	if StripeKey == "" {
		return nil, errors.New("STRIPE_KEY env var not found")
	}

	stripe.Key = StripeKey

	sub, err := subscription.Get(lic.SubscriptionID, nil)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
