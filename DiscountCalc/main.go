package main

import (
	"fmt"
    "./bookstore"
)

type OrderPriceTestCase struct {
	Order bookstore.Order
	Price float64
}

func getTestCases() []OrderPriceTestCase {
	return []OrderPriceTestCase {
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 1},{bookstore.Product{"book5", 8}, 1}}}, 51.2},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 1}}}, 8},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2}}}, 16},
		{bookstore.Order {Items: []bookstore.OrderItem {}}, 0},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 1},{bookstore.Product{"book2", 8}, 1}}}, 15.2},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 1},{bookstore.Product{"book2", 8}, 1},{bookstore.Product{"book3", 8}, 1}}}, 21.6},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 1},{bookstore.Product{"book2", 8}, 1},{bookstore.Product{"book3", 8}, 1},{bookstore.Product{"book4", 8}, 1}}}, 25.6},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 1},{bookstore.Product{"book2", 8}, 1},{bookstore.Product{"book3", 8}, 1},{bookstore.Product{"book4", 8}, 1}, {bookstore.Product{"book5", 8}, 1}}}, 30},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 1},{bookstore.Product{"book3", 8}, 1},{bookstore.Product{"book4", 8}, 2},{bookstore.Product{"book5", 8}, 2}}}, 51.2},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 1},{bookstore.Product{"book4", 8}, 1}}}, 40.8},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 2},{bookstore.Product{"book5", 8}, 1}}}, 55.6},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 2},{bookstore.Product{"book5", 8}, 2}}}, 60.0},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 2},{bookstore.Product{"book5", 8}, 2}, {bookstore.Product{"book1", 8}, 1}}}, 68.0},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 2},{bookstore.Product{"book5", 8}, 2}, {bookstore.Product{"book1", 8}, 1},{bookstore.Product{"book2", 8}, 1}}}, 75.2},
		{bookstore.Order {Items: []bookstore.OrderItem {{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 1},{bookstore.Product{"book5", 8}, 1},{bookstore.Product{"book1", 8}, 2},{bookstore.Product{"book2", 8}, 2},{bookstore.Product{"book3", 8}, 2},{bookstore.Product{"book4", 8}, 1},{bookstore.Product{"book5", 8}, 1}}}, 102.4},
	}
}

func getDiscountMap() map[int]float64 {
	return map[int]float64 {
		1: 0,
		2: 5,
		3: 10,
		4: 20,
		5: 25,
	}
}

func main() {
	discountMap := getDiscountMap()
	for _, testCase := range getTestCases() {
		calc := bookstore.BookSeriesPriceCalculator {&bookstore.BookSeriesDiscountManager {&testCase.Order, discountMap}}
		cart := bookstore.ShoppingCart {testCase.Order, &calc}
		calcedPrice := cart.GetPrice()
		fmt.Println(calcedPrice == testCase.Price, calcedPrice, testCase.Price)
	}
}


/*
сколько функций может лежать в пакете без ресивера
на сколько нужно заморачиваться с ссылочной передачей или копией
значимость апперкейса в структурах
тернарка
разбиение поведения и данных на разные структуры. структура без полей. структура-контейнер методов с неявным типом над которым производятся вычисления
использование короткой инициализации
*/