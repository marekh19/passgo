package views

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestButtonForwardsNativeFormAttributes(t *testing.T) {
	var output strings.Builder
	err := Button(Pay, templ.Attributes{
		"id": "pay", "type": "submit", "name": "intent", "value": "pay",
		"disabled": true, "aria-label": "Pay Sam", "data-intent": "payment",
	}).Render(context.Background(), &output)
	if err != nil {
		t.Fatal(err)
	}
	for _, attr := range []string{`id="pay"`, `type="submit"`, `name="intent"`, `value="pay"`, `disabled`, `aria-label="Pay Sam"`, `data-intent="payment"`} {
		if !strings.Contains(output.String(), attr) {
			t.Errorf("missing attribute %s", attr)
		}
	}
}

func TestLegacyLayoutDoesNotLoadDesignEnhancement(t *testing.T) {
	var output strings.Builder
	if err := Layout("Game").Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"payment.js", "pg-app", "pg-shell"} {
		if strings.Contains(output.String(), forbidden) {
			t.Errorf("legacy layout contains %s", forbidden)
		}
	}
}
