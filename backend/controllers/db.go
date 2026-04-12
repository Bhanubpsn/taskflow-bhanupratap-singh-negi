package controllers

import "github.com/jackc/pgx/v5/pgxpool"

var DB *pgxpool.Pool

// Since DB is being usde in multiple controllers, we can set it once and use it everywhere in controllers
// set in main
