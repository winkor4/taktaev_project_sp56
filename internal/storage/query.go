package storage

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users 
	(
		login TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS orders (
		user_login text NOT NULL,
		order_number TEXT NOT NULL,
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
