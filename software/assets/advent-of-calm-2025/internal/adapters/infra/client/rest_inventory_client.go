package client

import (
	"context"
	"fmt"
)

type RestInventoryClient struct{}

func (c *RestInventoryClient) CheckAndReserve(ctx context.Context, productID string, quantity int) (bool, error) {
	fmt.Printf("Calling Inventory Service REST API for product %s\n", productID)
	return true, nil
}
