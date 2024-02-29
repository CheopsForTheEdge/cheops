package main

import (
	"fmt"
	"log"
	"strconv"
)

type CounterManager struct {
	vals map[string]int
}

var Counter *CounterManager = &CounterManager{
	vals: make(map[string]int),
}

func (c *CounterManager) Get(id string) string {
	return fmt.Sprintf("%d", c.vals[id])
}

func (c *CounterManager) Handle(id, operation, value string) bool {
	valueInt, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("Value is not a valid int: %v\n", value)
		return false
	}

	if _, ok := c.vals[id]; !ok {
		c.vals[id] = 0
	}

	switch operation {
	case "insert":
		c.vals[id] = valueInt
	case "add":
		c.vals[id] += valueInt
	default:
		log.Printf("Invalid operation: %v\n", operation)
		return false
	}

	return true
}
