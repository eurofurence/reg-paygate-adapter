package integration

import (
	"context"
	"os"
	"testing"

	aulogging "github.com/StephanHCB/go-autumn-logging-zerolog"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database/inmemorydb"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/nexi"
	"github.com/stretchr/testify/require"
)

// TestNexiIntegrationFullFlow tests the full payment flow against the real Nexi API
// This test requires Nexi API credentials and will only run when NEXI_INTEGRATION_TEST=1
// environment variable is set. It tests create, query, delete payment link operations.
func TestNexiIntegrationFullFlow(t *testing.T) {
	if os.Getenv("NEXI_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping Nexi integration test-set NEXI_INTEGRATION_TEST=1 to run")
	}

	aulogging.SetupPlaintextLogging()
	db := inmemorydb.Create()
	database.SetRepository(db)

	// Use real Nexi API client (requires proper configuration)
	config.LoadTestingConfigurationFromPathOrAbort("../resources/integration-testconfig.yaml")
	require.NotEmpty(t, config.NexiDownstreamBaseUrl(), "Nexi downstream URL must be configured")
	require.NotEmpty(t, config.NexiInstanceApiSecret(), "Nexi API secret must be configured")
	require.NotEmpty(t, config.NexiMerchantNumber(), "Nexi merchant number must be configured")

	// Create the real client
	require.NoError(t, nexi.Create())
	client := nexi.Get()

	ctx := context.Background()

	// Test data
	paymentLinkRequest := nexi.NexiCreatePaymentRequest{
		Order: nexi.NexiOrder{
			Items: []nexi.NexiOrderItem{
				{
					Reference:        "INTEGRATION-TEST-REF",
					Name:             "Integration Test Payment",
					Quantity:         1.0,
					Unit:             "qty",
					UnitPrice:        1000, // 10.00 EUR
					TaxRate:          1900, // 19%
					TaxAmount:        159,
					GrossTotalAmount: 1000,
					NetTotalAmount:   841,
					ImageUrl:         nil,
				},
			},
			Amount:    1000,
			Currency:  "EUR",
			Reference: "INT-TEST-001",
		},
		Checkout: nexi.NexiCheckout{
			Url:             nil,
			IntegrationType: "hostedPaymentPage",
			ReturnUrl:       "https://example.com/success",
			CancelUrl:       "https://example.com/failure",
			Consumer: &nexi.NexiConsumer{
				Reference: "integration-test@example.com",
				Email:     "integration-test@example.com",
			},
			TermsUrl:         "",
			MerchantTermsUrl: nil,
			//ShippingCountries: []nexi.NexiCountry{
			//	{CountryCode: "DEU"},
			//},
			//Shipping: &nexi.NexiShipping{
			//	Countries: []nexi.NexiCountry{
			//		{CountryCode: "DEU"},
			//	},
			//	MerchantHandlesShippingCost: false,
			//	EnableBillingAddress:        true,
			//},
			//ConsumerType: &nexi.NexiConsumerType{
			//	Default:        "b2c",
			//	SupportedTypes: []string{"b2c", "b2b"},
			//},
			Charge:                      false,
			PublicDevice:                false,
			MerchantHandlesConsumerData: true,
			CountryCode:                 p("DEU"),
			Appearance: &nexi.NexiAppearance{
				DisplayOptions: nexi.NexiDisplayOptions{
					ShowMerchantName: true,
					ShowOrderSummary: true,
				},
				TextOptions: nexi.NexiTextOptions{
					CompletePaymentButtonText: "pay",
				},
			},
		},
		//Notifications: &nexi.NexiNotifications{
		//	Webhooks: []nexi.NexiWebhook{},
		//},
	}

	t.Run("create_payment_link", func(t *testing.T) {
		created, err := client.CreatePaymentLink(ctx, paymentLinkRequest)
		require.NoError(t, err, "Failed to create payment link")
		require.NotEmpty(t, created.ID, "Payment link ID should not be empty")
		require.NotEmpty(t, created.Link, "Payment link URL should not be empty")

		t.Logf("Created payment link: ID=%s, URL=%s", created.ID, created.Link)

		// Store the payment ID for subsequent tests
		paymentID := created.ID

		t.Run("query_payment_link", func(t *testing.T) {
			queried, err := client.QueryPaymentLink(ctx, paymentID)
			require.NoError(t, err, "Failed to query payment link")
			require.Equal(t, paymentID, queried.ID)
			require.Equal(t, paymentLinkRequest.Order.Reference, queried.ReferenceID)
			require.Equal(t, paymentLinkRequest.Order.Amount, queried.Amount)
			require.Equal(t, paymentLinkRequest.Order.Currency, queried.Currency)
			require.NotEmpty(t, queried.Link)

			t.Logf("Queried payment link: Status=%s, Amount=%d", queried.Status, queried.Amount)
		})

		t.Run("delete_payment_link", func(t *testing.T) {
			amount := paymentLinkRequest.Order.Amount
			err := client.DeletePaymentLink(ctx, paymentID, amount)
			require.NoError(t, err, "Failed to delete payment link")

			t.Logf("Deleted payment link: ID=%s", paymentID)

			// Verify deletion by attempting to query again (should fail)
			_, err = client.QueryPaymentLink(ctx, paymentID)
			// Note: Nexi API might not actually return an error for deleted payment links
			// The behavior varies, so we don't assert on this
			if err != nil {
				t.Logf("Query after delete returned error (expected): %v", err)
			} else {
				t.Logf("Query after delete succeeded (API may keep cancelled payments)")
			}
		})
	})
}

// TestNexiIntegrationErrorHandling tests error scenarios against the real Nexi API
func TestNexiIntegrationErrorHandling(t *testing.T) {
	if os.Getenv("NEXI_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping Nexi integration test - set NEXI_INTEGRATION_TEST=1 to run")
	}

	aulogging.SetupPlaintextLogging()
	db := inmemorydb.Create()
	database.SetRepository(db)

	config.LoadTestingConfigurationFromPathOrAbort("../resources/integration-testconfig.yaml")
	require.NotEmpty(t, config.NexiDownstreamBaseUrl(), "Nexi downstream URL must be configured")
	require.NotEmpty(t, config.NexiInstanceApiSecret(), "Nexi API secret must be configured")

	require.NoError(t, nexi.Create())
	client := nexi.Get()

	ctx := context.Background()

	t.Run("query_nonexistent_payment_link", func(t *testing.T) {
		_, err := client.QueryPaymentLink(ctx, "nonexistent-payment-id-12345")
		// Should return an error for non-existent payment link
		require.Error(t, err, "Querying non-existent payment link should fail")
		t.Logf("Expected error for non-existent payment: %v", err)
	})

	t.Run("delete_nonexistent_payment_link", func(t *testing.T) {
		err := client.DeletePaymentLink(ctx, "nonexistent-payment-id-12345", 0)
		// May or may not return an error depending on Nexi API behavior
		// We just log the result without requiring success or failure
		if err != nil {
			t.Logf("Expected error for deleting non-existent payment: %v", err)
		} else {
			t.Logf("Delete non-existent payment returned no error")
		}
	})
}

// TestNexiIntegrationWebhook tests webhook processing with real payment data
// This test creates a real payment link and simulates webhook notifications
func TestNexiIntegrationWebhookFlow(t *testing.T) {
	if os.Getenv("NEXI_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping Nexi integration test - set NEXI_INTEGRATION_TEST=1 to run")
	}

	t.Skip("Webhook integration test not implemented - requires manual webhook testing due to external dependencies")

	// FIXME: Not sure how to implement this yet - something along the lines of
	// 1. Create a real payment link
	// 2. Visit the payment URL to complete payment (or simulate)
	// 3. Wait for webhook notifications
	// 4. Verify webhook processing
	// needs to be tested
}

func p[T comparable](t T) *T {
	var nullValue T
	if t == nullValue {
		return nil
	}
	return &t
}
