package domain

import (
	"fmt"
)

type ShippingCalculator interface {
	Calculate(orderDetails OrderDetails) (float64, error)
}

type PostalShipping struct{}
type ExpressShipping struct{}
type DHLShipping struct{}
type DroneShipping struct{}

type OrderDetails struct {
	TotalWeightKg  float64
	DestinationZip string
}

const BaseRate = 10.5

func (s PostalShipping) Calculate(order OrderDetails) (float64, error) {
	cost := BaseRate + (order.TotalWeightKg * 1.5)
	if len(order.DestinationZip) > 0 && order.DestinationZip[0] == '0' {
		cost *= 0.9
	}
	return cost, nil
}

func (s ExpressShipping) Calculate(order OrderDetails) (float64, error) {
	cost := (BaseRate * 2.5) + (order.TotalWeightKg * 3.0)
	return cost, nil
}

func (s DHLShipping) Calculate(order OrderDetails) (float64, error) {
	cost := (BaseRate * 5.0) + (order.TotalWeightKg * 7.5) + 50.0
	return cost, nil
}

func (s DroneShipping) Calculate(order OrderDetails) (float64, error) {
	if order.TotalWeightKg > 5.0 {
		return 0, fmt.Errorf("weight limit exceeded for drone delivery (max 5kg)")
	}
	cost := 30.0 + (order.TotalWeightKg * 2.0)
	return cost, nil
}

func GetShippingCalculator(shippingType string) (ShippingCalculator, error) {
	switch shippingType {
	case "standard_mail":
		return PostalShipping{}, nil
	case "express_mail":
		return ExpressShipping{}, nil
	case "dhl_international":
		return DHLShipping{}, nil
	case "local_drone":
		return DroneShipping{}, nil
	default:
		return nil, fmt.Errorf("unknown shipping type: %s", shippingType)
	}
}

func CalculateShipping(shippingType string, order OrderDetails) (float64, error) {
	calculator, err := GetShippingCalculator(shippingType)
	if err != nil {
		return 0, err
	}
	return calculator.Calculate(order)
}

// ============= STATE ORDER =====================


