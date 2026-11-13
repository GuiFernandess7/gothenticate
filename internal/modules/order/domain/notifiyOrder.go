package domain

import (
	"GuiFernandess7/gothenticate/internal/pkg"
	"log"
)

type OrderObserver interface {
	OnStatusChanged(orderData *Order)
}

func (p *OrderState) notify(order *Order) {
	for _, o := range p.observers {
		o.OnStatusChanged(order)
	}
}

type NotificadorEstoque struct{}

func (n *NotificadorEstoque) OnStatusChanged(orderData *Order) {
	_, err := pkg.Call[*Order, struct{}](
		"POST",
		"https://estoque-go.free.beeceptor.com",
		"",
		orderData,
	)
	if err != nil {
		log.Printf("Failed to make request: %v", err)
		return
	}
	log.Printf("Estoque Notification sent successfully!")
}

type NotificadorFiscal struct{}

func (n *NotificadorFiscal) OnStatusChanged(orderData *Order) {
	_, err := pkg.Call[*Order, struct{}](
		"POST",
		"https://estoque-go.free.beeceptor.com",
		"",
		orderData,
	)
	if err != nil {
		log.Printf("Failed to make request: %v", err)
		return
	}
	log.Printf("Fiscal Notification sent successfully!")
}

type NotificadorEmail struct{}

func (n *NotificadorEmail) OnStatusChanged(orderData *Order) {
	_, err := pkg.Call[*Order, struct{}](
		"POST",
		"https://estoque-go.free.beeceptor.com",
		"",
		orderData,
	)
	if err != nil {
		log.Printf("Failed to make request: %v", err)
		return
	}
	log.Printf("Email sent successfully!")
}
