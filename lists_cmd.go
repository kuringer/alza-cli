package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kuringer/alza-cli/client"
)

func resolveFavoritesList(cl *client.TLSClient) (*client.CommodityList, error) {
	lists, err := cl.GetLists()
	if err != nil {
		return nil, err
	}

	names := []string{}
	if envName := strings.TrimSpace(os.Getenv("ALZA_FAVORITES_LIST")); envName != "" {
		names = append(names, envName)
	}
	names = append(names, "AGENT", "AGENTS")

	for _, name := range names {
		for _, l := range lists {
			if strings.EqualFold(l.Name, name) {
				return &l, nil
			}
		}
	}

	return nil, fmt.Errorf("favorites list not found (set ALZA_FAVORITES_LIST or create list named AGENT)")
}

// === FAVORITES ===

type FavoritesCmd struct {
	Show   FavoritesShowCmd   `cmd:"" default:"1" help:"Show favorites"`
	Add    FavoritesAddCmd    `cmd:"" help:"Add product to favorites"`
	Remove FavoritesRemoveCmd `cmd:"" help:"Remove product from favorites"`
}

type FavoritesShowCmd struct{}

func (c *FavoritesShowCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := resolveFavoritesList(cl)
	if err != nil {
		return err
	}

	items, err := cl.GetListItems(list.ID)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(items)
		return nil
	}

	if len(items) == 0 {
		fmt.Printf("List '%s' is empty\n", list.Name)
		return nil
	}

	fmt.Printf("List '%s' (%d items):\n\n", list.Name, len(items))
	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item.NavigationURL)
		if item.Price != "" {
			fmt.Printf("   Price: %s\n", item.Price)
		}
	}

	return nil
}

type FavoritesAddCmd struct {
	ProductID int `arg:"" help:"Product ID to add"`
}

func (c *FavoritesAddCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := resolveFavoritesList(cl)
	if err != nil {
		return err
	}

	if err := cl.AddToList(list.ID, c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Added product %d to list '%s'\n", c.ProductID, list.Name)
	return nil
}

type FavoritesRemoveCmd struct {
	ProductID int `arg:"" help:"Product ID to remove"`
}

func (c *FavoritesRemoveCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := resolveFavoritesList(cl)
	if err != nil {
		return err
	}

	if err := cl.RemoveFromList(list.ID, c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Removed product %d from list '%s'\n", c.ProductID, list.Name)
	return nil
}

// === LISTS ===

type ListsCmd struct {
	Show   ListsShowCmd   `cmd:"" default:"1" help:"Show all commodity lists"`
	Items  ListsItemsCmd  `cmd:"" help:"Show items in a specific list"`
	Create ListsCreateCmd `cmd:"" help:"Create a new list"`
	Add    ListsAddCmd    `cmd:"" help:"Add product to a list"`
}

type ListsShowCmd struct{}

func (c *ListsShowCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	lists, err := cl.GetLists()
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(lists)
		return nil
	}

	listTypes := map[int]string{
		0:  "custom",
		1:  "favorites",
		9:  "frequent",
		14: "buy-later",
		15: "pc-config",
	}

	fmt.Printf("Commodity Lists (%d):\n\n", len(lists))
	for _, l := range lists {
		typeStr := listTypes[l.Type]
		if typeStr == "" {
			typeStr = fmt.Sprintf("type-%d", l.Type)
		}
		fmt.Printf("  [%d] %s (%d items) - %s\n", l.ID, l.Name, l.ItemCount, typeStr)
	}

	return nil
}

type ListsItemsCmd struct {
	ListID int `arg:"" help:"List ID"`
}

func (c *ListsItemsCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	items, err := cl.GetListItems(c.ListID)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(items)
		return nil
	}

	if len(items) == 0 {
		fmt.Println("List is empty")
		return nil
	}

	fmt.Printf("List items (%d):\n\n", len(items))
	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item.NavigationURL)
		if item.Price != "" {
			fmt.Printf("   Price: %s\n", item.Price)
		}
	}

	return nil
}

type ListsCreateCmd struct {
	Name string `arg:"" help:"Name for the new list"`
}

func (c *ListsCreateCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	list, err := cl.CreateList(c.Name)
	if err != nil {
		return err
	}

	if g.Format == "json" {
		outputJSON(list)
		return nil
	}

	fmt.Printf("✓ List '%s' vytvorený (ID: %d)\n", list.Name, list.ID)
	return nil
}

type ListsAddCmd struct {
	ListID    int `arg:"" help:"List ID"`
	ProductID int `arg:"" help:"Product ID to add"`
}

func (c *ListsAddCmd) Run(g *Globals) error {
	cl, err := newClientWithAutoRefresh(g)
	if err != nil {
		return err
	}

	if err := cl.AddToList(c.ListID, c.ProductID); err != nil {
		return err
	}

	fmt.Printf("✓ Produkt %d pridaný do listu %d\n", c.ProductID, c.ListID)
	return nil
}
