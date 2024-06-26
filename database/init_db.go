// Copyright KubeArchive Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"os"
)

type flags struct {
	DatabaseName     string
	DatabaseUser     string
	DatabasePassword string
}

var defaultValues = &flags{
	DatabaseName:     "postgresdb",
	DatabaseUser:     "ps_user",
	DatabasePassword: "P0stgr3sdbP@ssword", // notsecret
}

const (
	host = "localhost"
	port = 5432
)

func main() {
	var flagValues flags
	flag.StringVar(&flagValues.DatabaseName, "database-name", defaultValues.DatabaseName, "PostgreSQL database name")
	flag.StringVar(&flagValues.DatabaseUser, "database-user", defaultValues.DatabaseUser, "PostgreSQL database user")
	flag.StringVar(&flagValues.DatabasePassword, "database-password", defaultValues.DatabasePassword, "PostgreSQL database password")
	flag.Parse()

	// connect to the DB.
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, flagValues.DatabaseUser, flagValues.DatabasePassword, flagValues.DatabaseName)

	// postgres is the driver type.
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// SQL instruction to create a table.
	sqlStatement := `
	CREATE TABLE IF NOT EXISTS public.test_objects (
		"id" serial PRIMARY KEY,
		"api_version" varchar NOT NULL,
		"kind" varchar NOT NULL,
		"name" varchar NOT NULL,
		"namespace" varchar NOT NULL,
		"resource_version" varchar NULL,
		"created_ts" timestamp NOT NULL,
		"updated_ts" timestamp NOT NULL,
		"data" jsonb NOT NULL
	);
	`
	_, err = db.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}
	fmt.Println("table test_objects created in the DB.")

	// load the test data from file
	testData := "test_objects.sql"
	//fmt.Println(testData)
	query, err := os.ReadFile(testData)
	if err != nil {
		panic(err)
	}

	// insert the data into the table.
	_, err = db.Exec(string(query))
	if err != nil {
		panic(err)
	}
	fmt.Println("testdata from test_objects.sql inserted in the DB.")
}
