package src

import (
	"errors"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/usagerecord"
)

func IncrementSubscriptionMeter(stripeKey string, lic *License, quantity int64) error {
	if stripeKey == "" {
		return errors.New("stripeKey is empty")
	}

	stripe.Key = stripeKey

	s, err := fetchStripeSubscription(stripeKey, lic)
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

func fetchStripeSubscription(stripeKey string, lic *License) (*stripe.Subscription, error) {
	if stripeKey == "" {
		return nil, errors.New("stripeKey is empty")
	}

	stripe.Key = stripeKey

	sub, err := subscription.Get(lic.SubscriptionID, nil)
	if err != nil {
		return nil, err
	}

	return sub, nil
}
