package bookstore

import (
	"sort"
)
type Order struct {
	Items []OrderItem
}

func (order *Order) Clone() *Order {
	clone := Order {}
	for _, item := range order.Items {
		clone.Items = append(clone.Items, item)
	}
	return &clone
}

func (order *Order) FindItem(title string) *OrderItem {
	var orderItem *OrderItem = nil
	for i, item := range order.Items {
		if item.Product.Title == title {
			orderItem = &order.Items[i]
			break
		}
	}
	return orderItem
}

func (order *Order) Sort() *Order {
	clone := order.Clone()
	sort.Slice(clone.Items, func(i, j int) bool {
		return clone.Items[i].Count > clone.Items[j].Count
	})
	return clone
}

func (order *Order) GetTotalItemsCount() int {
	count := 0
	for _, item := range order.Items {
		count += item.Count; 
	}
	return count
}

func (order *Order) GroupSameTitleItems() *Order {
	groupedOrder := Order {}
	for _, item := range order.Items {
		var existingItem *OrderItem = groupedOrder.FindItem(item.Product.Title)
		if existingItem == nil {
			groupedOrder.Items = append(groupedOrder.Items, item)
		} else {
			existingItem.Count += item.Count 
		}
	}
	return &groupedOrder
}