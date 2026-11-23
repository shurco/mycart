package queries

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/litepay"
)

// CustomerFilter narrows the admin customer list.
type CustomerFilter struct {
	// Search matches a substring of the address or the name, case-insensitively.
	Search string
	// RegisteredOnly drops the buyers who never created an account, leaving
	// only the rows an account-level action can be applied to.
	RegisteredOnly bool
	// Currency is the currency the spent totals are computed in. Carts keep the
	// currency they were created in, so a shop that changed currency has a
	// history in two of them; adding those minor units together would be
	// arithmetic on nothing. Totals therefore cover the given currency only,
	// and the count of purchases covers everything.
	Currency string
}

// likeContains renders a substring pattern for LIKE with the wildcards escaped,
// so that searching for "100%" looks for those four characters instead of
// matching every row. The callers pair it with `LIKE ? ESCAPE '\'`.
func likeContains(s string) string {
	escape := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + escape.Replace(strings.ToLower(s)) + "%"
}

// customerAddresses is one row per address, with the account (if any) hanging off it.
const customerAddresses = `
	FROM (
		SELECT email FROM customer
		UNION
		SELECT LOWER(TRIM(email)) AS email FROM cart WHERE TRIM(email) <> '' AND payment_status = ?
	) e
	LEFT JOIN customer c ON c.email = e.email
	`

// Customers returns one row per address the shop knows about, newest activity
// first.
//
// A "customer" here is an address, not an account: most buyers in this shop
// check out as guests, so the list is the union of the registered accounts and
// the addresses that have paid at least once. A row from the second group has
// no account behind it, which the caller can tell from the summary's
// Registered flag.
//
// Addresses are compared trimmed and lower-cased throughout. Checkout stores
// the address exactly as the buyer typed it, while registration normalises it,
// so "Anna@Example.com " and "anna@example.com" have to collapse into one row —
// the alternative is an operator seeing the same person twice and blocking the
// half that is not actually used to sign in. The same trimming is what keeps a
// cart whose address was typed as blanks out of the list: it collapses to "" and
// the guard drops it rather than showing a row with no address at all.
func (q *CustomerQueries) Customers(
	ctx context.Context, filter CustomerFilter, limit, offset int,
) ([]*models.CustomerSummary, int, error) {
	where, filterParams := customerFilterClause(filter)

	total, err := q.countCustomers(ctx, where, filterParams)
	if err != nil {
		return nil, 0, err
	}

	query := q.customerSummaryQuery(where)

	// The order of these parameters is the textual order of the placeholders:
	// the paid guard inside the address union, then the two in the totals join,
	// then the search patterns of the WHERE clause, then pagination.
	rowParams := append([]any{litepay.PAID, filter.Currency, litepay.PAID}, filterParams...)
	rowParams = append(rowParams, limit, offset)

	rows, err := q.DB.QueryContext(ctx, query, rowParams...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	customers := []*models.CustomerSummary{}
	for rows.Next() {
		summary, err := scanCustomerSummary(rows, filter.Currency)
		if err != nil {
			return nil, 0, err
		}

		customers = append(customers, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

// customerFilterClause renders the WHERE clause of a filter and its parameters.
// filterParams holds the search patterns alone, so the two callers can put them
// after their own parameters.
func customerFilterClause(filter CustomerFilter) (where string, filterParams []any) {
	var conditions []string
	if filter.Search != "" {
		conditions = append(conditions,
			`(e.email LIKE ? ESCAPE '\' OR LOWER(COALESCE(c.name, '')) LIKE ? ESCAPE '\')`)
		pattern := likeContains(filter.Search)
		filterParams = append(filterParams, pattern, pattern)
	}
	if filter.RegisteredOnly {
		conditions = append(conditions, "c.id IS NOT NULL")
	}

	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	return where, filterParams
}

// countCustomers counts the addresses a filter matches.
func (q *CustomerQueries) countCustomers(ctx context.Context, where string, filterParams []any) (int, error) {
	// The count reads the addresses and their accounts only. Joining the
	// purchase totals here would make it GROUP BY the whole cart table to
	// produce a number it never looks at.
	addressParams := append([]any{litepay.PAID}, filterParams...)
	var total int
	if err := q.DB.QueryRowContext(ctx,
		"SELECT COUNT(*)"+customerAddresses+where, addressParams...).Scan(&total); err != nil {
		return 0, err
	}

	return total, nil
}

// customerSummaryQuery builds the page query of Customers: the summary columns
// over the addresses, their accounts and their purchase totals, ordered and
// paged.
func (q *CustomerQueries) customerSummaryQuery(where string) string {
	epochCreated := q.DB.Dialect().Epoch("c.created")
	epochUpdated := q.DB.Dialect().Epoch("c.updated")

	// One row per address again, so this join cannot multiply the one above.
	totals := fmt.Sprintf(`
	LEFT JOIN (
		SELECT
			LOWER(TRIM(email)) AS email,
			COUNT(*) AS purchases,
			SUM(CASE WHEN currency = ? THEN amount_total ELSE 0 END) AS spent,
			MAX(%s) AS last_order
		FROM cart
		WHERE payment_status = ? AND TRIM(email) <> ''
		GROUP BY LOWER(TRIM(email))
	) o ON o.email = e.email
	`, q.DB.Dialect().Epoch("created"))

	// Only the column list goes through Sprintf. The FROM clause is
	// concatenated onto it rather than interpolated into the format string,
	// because the epoch fragments it carries are themselves of the form
	// strftime('%s', …): a percent sign that reaches a format string is not a
	// syntax error, it silently consumes the next argument and leaves the
	// query quietly answering a different question.
	//
	// account_id is the raw join key, deliberately not coalesced: NULL is how a
	// row says "this address never registered", which is the one thing the
	// caller has to know before offering an account-level action.
	columns := fmt.Sprintf(`
	SELECT
		c.id AS account_id,
		e.email,
		COALESCE(c.name, '') AS name,
		c.active,
		%s AS created,
		%s AS updated,
		COALESCE(o.purchases, 0) AS purchases,
		COALESCE(o.spent, 0) AS spent,
		COALESCE(o.last_order, 0) AS last_order
	`, epochCreated, epochUpdated)

	// Buyers first, then addresses that only ever registered, and within each
	// group the most recent purchase first. The leading (… IS NULL) sort key is
	// what keeps the two engines' opposite NULL placement out of the result:
	// ordering on the timestamp alone would put one group at the wrong end on
	// one of them.
	order := "\n\tORDER BY (o.last_order IS NULL), o.last_order DESC, e.email\n"

	return columns + customerAddresses + totals + where + order + " LIMIT ? OFFSET ?"
}

// scanCustomerSummary reads one row of the Customers listing.
func scanCustomerSummary(rows *sql.Rows, currency string) (*models.CustomerSummary, error) {
	var accountID, name sql.NullString
	var active sql.NullBool
	var created, updated, lastOrder sql.NullInt64
	summary := &models.CustomerSummary{}

	if err := rows.Scan(
		&accountID,
		&summary.Email,
		&name,
		&active,
		&created,
		&updated,
		&summary.Purchases,
		&summary.Spent,
		&lastOrder,
	); err != nil {
		return nil, err
	}

	summary.ID = accountID.String
	summary.Registered = accountID.Valid
	summary.Name = name.String
	summary.Active = active.Bool
	summary.Created = created.Int64
	summary.Updated = updated.Int64
	summary.LastOrder = lastOrder.Int64
	// The totals are in this currency by construction — the query only adds
	// up carts that carry it — so the row has to say which one that was, or
	// the operator is looking at a number with no unit.
	summary.Currency = currency

	return summary, nil
}
