package stripe

import (
	"fmt"
	"os"

	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/customer"
	"github.com/stripe/stripe-go/v80/paymentintent"
	"github.com/stripe/stripe-go/v80/price"
	"github.com/stripe/stripe-go/v80/product"
)

func init() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

type StripeService struct {
	apiKey string
}

func NewStripeService() *StripeService {
	return &StripeService{
		apiKey: os.Getenv("STRIPE_SECRET_KEY"),
	}
}

func (s *StripeService) CreateCustomer(email, name string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(name),
	}
	
	return customer.New(params)
}

func (s *StripeService) CreateProduct(name, description string) (*stripe.Product, error) {
	params := &stripe.ProductParams{
		Name:        stripe.String(name),
		Description: stripe.String(description),
	}
	
	return product.New(params)
}

func (s *StripeService) CreatePrice(productID string, unitAmount int64, currency string) (*stripe.Price, error) {
	params := &stripe.PriceParams{
		Product:     stripe.String(productID),
		UnitAmount:  stripe.Int64(unitAmount),
		Currency:    stripe.String(currency),
	}
	
	return price.New(params)
}

func (s *StripeService) CreatePaymentIntent(amount int64, currency string, customerID string) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		Customer: stripe.String(customerID),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	
	return paymentintent.New(params)
}

func (s *StripeService) GetCustomer(customerID string) (*stripe.Customer, error) {
	return customer.Get(customerID, nil)
}

func (s *StripeService) ListCustomers(limit int64) ([]*stripe.Customer, error) {
	params := &stripe.CustomerListParams{}
	params.Limit = stripe.Int64(limit)
	
	var customers []*stripe.Customer
	i := customer.List(params)
	for i.Next() {
		customers = append(customers, i.Customer())
	}
	
	if err := i.Err(); err != nil {
		return nil, fmt.Errorf("error listing customers: %w", err)
	}
	
	return customers, nil
}