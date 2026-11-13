package order_test

import (
	"testing"

	domain "GuiFernandess7/gothenticate/internal/modules/order/domain"
)

func TestCalculateShipping_Postal(t *testing.T) {
	cost, err := domain.CalculateShipping("standard_mail", domain.OrderDetails{TotalWeightKg: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := domain.BaseRate + (2 * 1.5)
	if cost != want {
		t.Fatalf("expected cost %v, got %v", want, cost)
	}
}

func TestCalculateShipping_Postal_DiscountForZipStartingWithZero(t *testing.T) {
	cost, err := domain.CalculateShipping("standard_mail", domain.OrderDetails{TotalWeightKg: 2, DestinationZip: "01310-100"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := (domain.BaseRate + (2 * 1.5)) * 0.9
	if cost != want {
		t.Fatalf("expected discounted cost %v, got %v", want, cost)
	}
}

func TestCalculateShipping_Express(t *testing.T) {
	cost, err := domain.CalculateShipping("express_mail", domain.OrderDetails{TotalWeightKg: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := (domain.BaseRate * 2.5) + (1 * 3.0)
	if cost != want {
		t.Fatalf("expected cost %v, got %v", want, cost)
	}
}

func TestCalculateShipping_DHL(t *testing.T) {
	cost, err := domain.CalculateShipping("dhl_international", domain.OrderDetails{TotalWeightKg: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := (domain.BaseRate * 5.0) + (1 * 7.5) + 50.0
	if cost != want {
		t.Fatalf("expected cost %v, got %v", want, cost)
	}
}

func TestCalculateShipping_Drone_WithinLimit(t *testing.T) {
	cost, err := domain.CalculateShipping("local_drone", domain.OrderDetails{TotalWeightKg: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 30.0 + (5 * 2.0)
	if cost != want {
		t.Fatalf("expected cost %v, got %v", want, cost)
	}
}

func TestCalculateShipping_Drone_ExceedsWeightLimit(t *testing.T) {
	_, err := domain.CalculateShipping("local_drone", domain.OrderDetails{TotalWeightKg: 5.1})
	if err == nil {
		t.Fatal("expected an error for drone delivery above the 5kg limit")
	}
}

func TestCalculateShipping_UnknownCarrier(t *testing.T) {
	_, err := domain.CalculateShipping("teleport", domain.OrderDetails{TotalWeightKg: 1})
	if err == nil {
		t.Fatal("expected an error for an unknown shipping type")
	}
}
