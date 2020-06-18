package bookstore

type OrderPriceCalculator interface {
	Cost(order *Order) float64
}

type ShoppingCart struct {
	Order Order
	PriceCalculator OrderPriceCalculator
}

func (cart *ShoppingCart) GetPrice() float64 {
	return cart.PriceCalculator.Cost(&cart.Order)
}
