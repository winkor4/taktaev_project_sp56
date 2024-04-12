package mock

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS mock_orders (
		order_number TEXT UNIQUE NOT NULL,
		status TEXT NOT NULL,
		sum INTEGER NOT NULL
	);`,
}

var (
	queryNewOrders = `
	SELECT
		orders.order_number
	FROM
		orders
		LEFT JOIN mock_orders 
		ON orders.order_number = mock_orders.order_number
	WHERE
		orders.status = 'NEW'
		AND mock_orders.status IS NULL`

	queryRegister = `
	INSERT INTO mock_orders
	(
		order_number,
		status,
		sum
	)
	VALUES 
	(
		$1,
		$2,
		$3
	)
	ON CONFLICT (order_number) DO NOTHING;`

	queryGetOrder = `
	SELECT
		order_number,
		status,
		sum
	FROM
		mock_orders
	WHERE
		order_number = $1`
)
