package main

import "sync"

type Item interface {
	Price(quantity uint) uint
	PreProcess() error
	Process() error
}

type qItem struct {
	quantity uint
	Item
}

type Discount interface {
}

type Voucher interface {
}

type Basket struct {
	mu        sync.RWMutex
	Items     []qItem
	Vouchers  map[string]Voucher
	Discounts []Discount
}

func (b *Basket) ItemAdd(i Item, q uint) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for n := range b.Items {
		if b.Items[n].Item == i {
			b.Items[n].quantity += q
			return true
		}
	}
	b.Items = append(b.Items, qItem{
		quantity: q,
		Item:     i,
	})
	return false
}

func (b *Basket) ItemQuantity(i Item, q uint) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for n := range b.Items {
		if b.Items[n].Item == i {
			b.Items[n].quantity = q
			return true
		}
	}
	return false
}

func (b *Basket) ItemRemove(i Item) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for n := range b.Items {
		if b.Items[n].Item == i {
			b.Items, b.Items[len(b.Items)-1] = append(b.Items[:n], b.Items[n+1:]...), aItem{}
			return true
		}
	}
	return false
}
