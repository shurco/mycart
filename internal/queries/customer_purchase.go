package queries

import (
	"context"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
)

// CustomerPurchases returns the paid orders of an address, newest first, each
// with its lines and the digital goods those lines hand over.
//
// Only paid orders: a cart that was never paid for is not a purchase, and this
// is the list the shop calls the buyer's own. The address is compared
// normalized, because checkout keeps whatever was typed while the account
// stores the lower-cased form — otherwise a buyer who typed a capital letter
// would find an empty cabinet.
func (q *CustomerQueries) CustomerPurchases(ctx context.Context, email string) ([]*models.CustomerPurchase, error) {
	carts, err := NewBase(q.DB).PaidCartsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Three reads for the whole cabinet rather than two per line. An order list
	// asks the same question of every entry in it, and each of these is one
	// query over the ids the orders mention, so the page costs the same whether
	// the buyer has one order or twenty.
	products, err := q.purchaseProducts(ctx, cartProducts(carts))
	if err != nil {
		return nil, err
	}

	files, err := q.purchaseFiles(ctx, productsOfKind(products, models.DigitalFile))
	if err != nil {
		return nil, err
	}

	codes, err := q.purchaseCodes(ctx, cartIDs(carts))
	if err != nil {
		return nil, err
	}

	purchases := []*models.CustomerPurchase{}
	for _, cart := range carts {
		purchases = append(purchases, &models.CustomerPurchase{
			ID:          cart.ID,
			Created:     cart.Created,
			AmountTotal: cart.AmountTotal,
			Currency:    cart.Currency,
			Items:       purchaseItems(cart, products, files, codes),
		})
	}

	return purchases, nil
}

// cartProducts lists every product the given orders mention, each once.
func cartProducts(carts []*models.Cart) []string {
	seen := map[string]bool{}
	ids := []string{}
	for _, cart := range carts {
		for _, line := range cart.Cart {
			if !seen[line.ProductID] {
				seen[line.ProductID] = true
				ids = append(ids, line.ProductID)
			}
		}
	}

	return ids
}

// productsOfKind narrows a set of products to those delivered one way.
func productsOfKind(products map[string]*models.Product, digital string) []string {
	ids := []string{}
	for id, product := range products {
		if product.Digital.Type == digital {
			ids = append(ids, id)
		}
	}

	return ids
}

// cartIDs lists the ids of the given orders.
func cartIDs(carts []*models.Cart) []string {
	ids := make([]string, 0, len(carts))
	for _, cart := range carts {
		ids = append(ids, cart.ID)
	}

	return ids
}

// CustomerOwnsProduct reports whether an address has ever paid for a product.
//
// The lines of each cart are parsed here rather than matched in SQL. A cart's
// contents live in a text column holding JSON, and the only comparison SQL
// offers over it is a substring one — a match on a string, not on the id field,
// which is both weaker than it looks and spelled differently on the two
// engines. A buyer has a handful of orders, so reading them costs nothing and
// answers the question exactly.
func (q *CustomerQueries) CustomerOwnsProduct(ctx context.Context, email, productID string) (bool, error) {
	carts, err := NewBase(q.DB).PaidCartsByEmail(ctx, email)
	if err != nil {
		return false, err
	}

	for _, cart := range carts {
		for _, line := range cart.Cart {
			if line.ProductID == productID {
				return true, nil
			}
		}
	}

	return false, nil
}

// EntitledDigitalFile returns a digital file if the address has paid for the
// product it belongs to, and errors.ErrProductNotFound otherwise.
//
// The one error answers "no such file" and "that one is not yours" alike, and
// that is the point: told apart, the endpoint becomes a way to walk the shop's
// catalogue of guides by id and learn what exists.
func (q *CustomerQueries) EntitledDigitalFile(ctx context.Context, email, fileID string) (*models.File, error) {
	file, productID, err := q.digitalFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	owned, err := q.CustomerOwnsProduct(ctx, email, productID)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, errors.ErrProductNotFound
	}

	return file, nil
}

// purchaseItems resolves the lines of one order against what has already been
// read.
//
// A line whose product no longer exists is dropped rather than shown blank: the
// cart stores ids only, so once the row is gone there is no name left to
// display, and a nameless line with nothing behind it is worse than an order
// that reads one item shorter. The order itself stays, with its total.
func purchaseItems(
	cart *models.Cart,
	products map[string]*models.Product,
	files map[string][]models.CustomerPurchaseFile,
	codes map[string][]string,
) []models.CustomerPurchaseItem {
	items := []models.CustomerPurchaseItem{}

	for _, line := range cart.Cart {
		product, ok := products[line.ProductID]
		if !ok {
			continue
		}

		item := models.CustomerPurchaseItem{
			ProductID: product.ID,
			Name:      product.Name,
			Slug:      product.Slug,
			Quantity:  line.Quantity,
			Digital:   product.Digital.Type,
		}

		switch product.Digital.Type {
		case models.DigitalFile:
			item.Files = files[product.ID]
		case models.DigitalData:
			item.Codes = codes[codeKey(cart.ID, product.ID)]
		}

		items = append(items, item)
	}

	return items
}

// codeKey names one order's claim on one product, which is what a licence key
// belongs to.
func codeKey(cartID, productID string) string {
	return cartID + "\x00" + productID
}
