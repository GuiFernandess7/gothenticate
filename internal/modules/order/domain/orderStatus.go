package domain

import "GuiFernandess7/gothenticate/internal/infra/errors"

// ENUM
type OrderStatus string

const (
	StatusOrderPending    OrderStatus = "pending"
	StatusOrderPaid       OrderStatus = "paid"
	StatusOrderPreparing  OrderStatus = "preparing"
	StatusOrderSent		  OrderStatus = "sent"
	StatusOrderReceived   OrderStatus = "received"
	StatusOrderCanceled   OrderStatus = "canceled"
)

// INTERFACE
type OrderStateInterface interface {
	Label() OrderStatus
	Pagar(state *OrderState, order *Order) error
    Enviar(state *OrderState, order *Order) error
    ConfirmarEntrega(state *OrderState, order *Order) error
    Cancelar(state *OrderState, order *Order) error
}

// MAIN STRUCT
type OrderState struct {
	status OrderStateInterface
	observers []OrderObserver
}

// MAIN METHODS
func (o *OrderState) Label() OrderStatus {
	return o.status.Label()
}

func (o *OrderState) Pagar(order *Order) error {
	return o.status.Pagar(o, order)
}

func (o *OrderState) Enviar(order *Order) error {
	return o.status.Enviar(o, order)
}

func (o *OrderState) ConfirmarEntrega(order *Order) error {
	return o.status.ConfirmarEntrega(o, order)
}

func (o *OrderState) Cancelar(order *Order) error {
	return o.status.Cancelar(o, order)
}

// METHOD
func NewOrderStatus(label OrderStatus, observers ...OrderObserver) *OrderState {
	return &OrderState{
		status:    stateFromLabel(label),
		observers: observers,
	}
}

func stateFromLabel(label OrderStatus) OrderStateInterface {
	switch label {
	case StatusOrderPending:
		return &StatusPending{}
	case StatusOrderPaid:
		return &StatusPaid{}
	case StatusOrderPreparing:
		return &StatusPreparing{}
	case StatusOrderSent:
		return &StatusSent{}
	case StatusOrderReceived:
		return &StatusReceived{}
	default:
		return &StatusPending{}
	}
}

// STATUS STRUCT AND METHODS
type StatusPending struct {}

func (o *StatusPending) Label() OrderStatus {
	return StatusOrderPending
}

func (o *StatusPending) Pagar(state *OrderState, order *Order) error {
	state.status = &StatusPaid{}
	state.notify(order)
	return nil
}

func (o *StatusPending) Enviar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusPending) ConfirmarEntrega(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusPending) Cancelar(state *OrderState, order *Order) error {
	state.status = &StatusCanceled{}
	state.notify(order)
	return nil
}

type StatusPaid struct {}

func (o *StatusPaid) Label() OrderStatus {
	return StatusOrderPaid
}

func (o *StatusPaid) Pagar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusPaid) Enviar(state *OrderState, order *Order) error {
	state.status = &StatusPreparing{}
	state.notify(order)
	return nil
}

func (o *StatusPaid) ConfirmarEntrega(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusPaid) Cancelar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

type StatusPreparing struct {}

func (o *StatusPreparing) Label() OrderStatus {
	return StatusOrderPreparing
}

func (o *StatusPreparing) Pagar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusPreparing) Enviar(state *OrderState, order *Order) error {
	state.status = &StatusSent{}
	state.notify(order)
	return nil
}

func (o *StatusPreparing) ConfirmarEntrega(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusPreparing) Cancelar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

type StatusSent struct {}

func (o *StatusSent) Label() OrderStatus {
	return StatusOrderSent
}

func (o *StatusSent) Pagar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusSent) Enviar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusSent) ConfirmarEntrega(state *OrderState, order *Order) error {
	state.status = &StatusReceived{}
	state.notify(order)
	return nil
}

func (o *StatusSent) Cancelar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

type StatusReceived struct {}

func (o *StatusReceived) Label() OrderStatus {
	return StatusOrderReceived
}

func (o *StatusReceived) Pagar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusReceived) Enviar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusReceived) ConfirmarEntrega(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusReceived) Cancelar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

type StatusCanceled struct {}

func (o *StatusCanceled) Label() OrderStatus {
	return StatusOrderCanceled
}

func (o *StatusCanceled) Pagar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusCanceled) Enviar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusCanceled) ConfirmarEntrega(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}

func (o *StatusCanceled) Cancelar(state *OrderState, order *Order) error {
	return errors.ErrInvalidOperation
}
