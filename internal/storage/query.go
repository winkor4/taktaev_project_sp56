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
		sum INTEGER NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS bonuses (
		user_login text NOT NULL,
		sum INTEGER NOT NULL,
		out INTEGER NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS spending (
		user_login text NOT NULL,
		order_number text NOT NULL,
		sum INTEGER NOT NULL,
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

	queryInsertdOrder = `
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
)
