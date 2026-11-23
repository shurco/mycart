package queries

// paginate appends LIMIT and OFFSET placeholders to query and their values to
// params. A limit of zero or less leaves the query unpaged, and an offset
// without a limit is ignored.
func paginate(query string, params []any, limit, offset int) (string, []any) {
	if limit > 0 {
		query += " LIMIT ?"
		params = append(params, limit)
		if offset > 0 {
			query += " OFFSET ?"
			params = append(params, offset)
		}
	}
	return query, params
}
