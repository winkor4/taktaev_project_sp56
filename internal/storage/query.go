package storage

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users 
	(
		login TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS orders (
		user_login text NOT NULL,
		order_number TEXT UNIQUE NOT NULL,
		date TIMESTAMP NOT NULL,
		status TEXT NOT NULL,
		sum FLOAT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS bonuses (
		user_login text NOT NULL,
		sum FLOAT NOT NULL,
		out FLOAT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS spending (
		user_login text NOT NULL,
		order_number text NOT NULL,
		sum FLOAT NOT NULL,
		date TIMESTAMP NOT NULL
	);`,
}

var (
	queryRegister = `
	INSERT INTO users 
	(
		login, 
		password
	)
	VALUES 
	(
		$1, 
		$2
	)
	ON CONFLICT (login) DO NOTHING;`

	queryPassword = `
	SELECT
		password
	FROM
		users
	WHERE 
		login = $1`

	queryInsertOrder = `
	INSERT INTO orders
	(
		user_login,
		order_number,
		date,
		status,
		sum
	)
	VALUES 
	(
		$1,
		$2,
		$3,
		$4,
		$5
	)
	ON CONFLICT (order_number) DO NOTHING;`

	querySelectOrder = `
	SELECT
		user_login
	FROM
		orders
	WHERE
		order_number = $1`

	querySelectOrders = `
	SELECT
		order_number,
		date,
		status,
		sum
	FROM
		orders
	WHERE
		user_login = $1
	ORDER BY
		date`

	queryOrdersToRefresh = `
	SELECT
		order_number
	FROM
		orders
	WHERE
		NOT status IN ('INVALID', 'PROCESSED')`

	queryUpdateOrder = `
	UPDATE orders
	SET
		status = $1,
		sum = $2
	WHERE
		order_number = $3`

	queryLogins = `
	SELECT
		user_login,
		order_number
	FROM
		orders
	WHERE
		order_number IN ($1)`

	queryInsertBonuses = `
	INSERT INTO bonuses
	(
		user_login,
		sum,
		out
	)
	VALUES
	(
		$1,
		$2,
		$3
	)`

	queryBalance = `
	SELECT
		SUM(sum),
		SUM(out)
	FROM
		bonuses
	WHERE
		user_login = $1`

	queryInsertSpending = `
	INSERT INTO spending
	(
		user_login,
		order_number,
		date,
		sum
	)
	VALUES 
	(
		$1,
		$2,
		$3,
		$4
	)`

	querySelectSpending = `
	SELECT
		order_number,
		sum,
		date
	FROM
		spending
	WHERE
		user_login = $1
	ORDER BY
		date`
)
