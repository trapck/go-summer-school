package bookstore

import (
	"math"
)

type DiscountManager interface {
	CombineMaxDiscountOrder() DiscountCombinedOrder
}

type BookSeriesPriceCalculator struct {
	DiscountManager DiscountManager	
}

func (calculator *BookSeriesPriceCalculator) Cost(order *Order) float64 {
	price := 0.0
	if calculator.DiscountManager != nil {
		price = calculator.CalcWithDiscount(order, calculator.DiscountManager)
	} else {
		price = calculator.CalcRaw(order)
	}
	return math.Round(price * 100) / 100
}

func (calculator *BookSeriesPriceCalculator) CalcWithDiscount(order *Order, discountManager DiscountManager) float64 {
	price := 0.0
	var discountCombinedOrder DiscountCombinedOrder = discountManager.CombineMaxDiscountOrder()
	for _, discountGroup := range discountCombinedOrder.DiscountGroups {
		price += calculator.CalcDiscountGroupPrice(&discountGroup)
	}
	return price
}

func (calculator *BookSeriesPriceCalculator) CalcRaw(order *Order) float64 {
	price := 0.0
	for _, item := range order.Items {
		price += float64(item.Count) * item.Product.Price 
	}
	return price
}

func (calculator *BookSeriesPriceCalculator) CalcDiscountGroupPrice(discountGroup *DiscountGroup) float64 {
	price := 0.0
	coef := 1 - discountGroup.Discount / 100
	for _, product := range discountGroup.Products {
		price += product.Price * coef
	}
	return price
}