package bookstore

type BookSeriesDiscountManager struct {
	Order *Order
	DiscountMap map[int]float64
}

func (manager *BookSeriesDiscountManager) CombineMaxDiscountOrder() DiscountCombinedOrder {
	combinedOrders := []DiscountCombinedOrder {}
	for i := range manager.Order.Items {
		combinedOrders = append(combinedOrders, manager.CombineDiscountOrder(i + 1))
	}
	return manager.FindMaxDiscountCombinedOrder(combinedOrders)
}

func (manager *BookSeriesDiscountManager) CombineDiscountOrder(maxChunk int) DiscountCombinedOrder {
	order := manager.Order.Clone().GroupSameTitleItems().Sort()
	combinedOrder := DiscountCombinedOrder {}
	totalOrderItemsCount := order.GetTotalItemsCount()
	for totalOrderItemsCount > 0 {
		discountGroup := DiscountGroup {}
		for i, item := range order.Items {
			if len(discountGroup.Products) < maxChunk && item.Count != 0 {
				discountGroup.Products = append(discountGroup.Products, item.Product)
				order.Items[i].Count = item.Count - 1
				totalOrderItemsCount--
			}
		}
		discountGroup.Discount = manager.GetDiscountForProductsGroup(discountGroup.Products)
		combinedOrder.DiscountGroups = append(combinedOrder.DiscountGroups, discountGroup)
	}
	return combinedOrder
}

func (manager *BookSeriesDiscountManager) FindMaxDiscountCombinedOrder(discountCombinedOrders []DiscountCombinedOrder) DiscountCombinedOrder {
	currentDiscuont, maxDiscount := 0.0, 0.0
	var maxDiscountCombinedOrder DiscountCombinedOrder
	for _, combinedOrder := range discountCombinedOrders {
		for _, group := range combinedOrder.DiscountGroups {
			currentDiscuont += group.Discount
		}
		if currentDiscuont >= maxDiscount {
			maxDiscount = currentDiscuont
			maxDiscountCombinedOrder = combinedOrder
		}
		currentDiscuont = 0
	}
	return maxDiscountCombinedOrder
}

func (manager *BookSeriesDiscountManager) GetDiscountForProductsGroup(products []Product) float64 {
	var result float64
	if discount, ok := manager.DiscountMap[len(products)]; ok == true {
		result = discount
	}
	return result
}