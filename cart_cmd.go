package main

import (
	"fmt"
)

// === CART ===

type CartCmd struct {
	Show   CartShowCmd   `cmd:"" default:"1" help:"Show cart contents"`
	Add    CartAddCmd    `cmd:"" help:"Add product to cart"`
	Remove CartRemoveCmd `cmd:"" help:"Remove product from cart"`
	Clear  CartClearCmd  `cmd:"" help:"Clear entire cart"`
}

type CartShowCmd struct{}

func (c *CartShowCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	items, err := cl.GetCart()
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(items)
		return nil
	}

	if len(items) == 0 {
		fmt.Println("Cart is empty")
		return nil
	}

	fmt.Printf("Cart (%d items):\n\n", len(items))
	for i, item := range items {
		if item.Name != "" {
			fmt.Printf("%d. [%d] %s\n", i+1, item.ProductID, item.Name)
			fmt.Printf("   Price: %s | Qty: %d\n", item.Price, item.Count)
			if item.URL != "" {
				fmt.Printf("   %s\n", item.URL)
			}
			fmt.Println()
		} else {
			fmt.Printf("%d. Product ID: %d (qty: %d)\n", i+1, item.ProductID, item.Count)
		}
	}

	return nil
}

type CartAddCmd struct {
	ProductID int `arg:"" help:"Product ID to add"`
	Quantity  int `help:"Quantity" default:"1" short:"q"`
}

func (c *CartAddCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.AddToCart(c.ProductID, c.Quantity); err != nil {
		return err
	}

	fmt.Printf("✓ Added product %d to cart (qty: %d)\n", c.ProductID, c.Quantity)
	return nil
}

type CartRemoveCmd struct {
	ProductID int `arg:"" help:"Product ID to remove"`
}

func (c *CartRemoveCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.RemoveFromCart(c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Removed product %d from cart\n", c.ProductID)
	return nil
}

type CartClearCmd struct{}

func (c *CartClearCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.ClearCart(); err != nil {
		return err
	}

	fmt.Println("✓ Cart cleared")
	return nil
}
