package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4"

	"os"
)

//////////////////////////////////////////////////////
// SCHEMA & TABLE & COLUMN CREATION FUNCTIONS       //
//////////////////////////////////////////////////////

// AddDatabase adds a new database to the server, second param is the database name.
func AddDatabase(db *sql.DB, database string) {
	databaseCreateQuery := fmt.Sprintf("create database %s", database)
	_, err := db.Exec(databaseCreateQuery)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created database %s! \n", database)
}

// AddSchema  adds schema, table, and columns that are needed to the database. Column names and datatypes are asked in every iteration.
// You can check https://www.w3schools.com/sql/sql_datatypes.asp for available datatypes.
//
// Example: AddSchema(database, yarak_schema, kürek_table), where yarak_schema is the schema name and kürek_table is the table name.
func AddSchema(db *sql.DB, schema string, tableName string) {
	// Check if database is alive.
	err := db.PingContext(context.Background())
	if err != nil {
		panic(err)
	}
	var colname string
	var coltype string

	fmt.Printf("Creating schema %s... \n", schema)
	schemaCreateQuery := fmt.Sprintf("create schema %s", schema)
	fmt.Println(schemaCreateQuery)
	_, err = db.Exec(schemaCreateQuery)
	fmt.Printf("Created schema %s! \n", schema)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Creating table %s...", tableName)
	fmt.Println("Enter the first column name: ")
	_, err = fmt.Scanln(&colname)
	if err != nil {
		panic(err)
	}
	fmt.Println("Enter the first column type: ")
	_, err = fmt.Scanln(&coltype)
	if err != nil {
		panic(err)
	}
	AddTable(db, schema, tableName, colname, coltype)
}

//AddTable adds new table with single column to the database.
//
// Example: AddTable(database, "yarak_schema", "kürek_table", "kazancı","bedih")
//
// Yarak_schema is the schema name, kürek_table is the table name, kazancı is the initial column name and bedih is the initial column type.
//
// You can check https://www.w3schools.com/sql/sql_datatypes.asp for available datatypes.
func AddTable(db *sql.DB, schema string, table string, colname string, coltype string) {
	tableCreateQuery := fmt.Sprintf("create table %s.%s \n(\n %s %s \n)", schema, table, colname, coltype)
	_, err := db.Exec(tableCreateQuery)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created table %s! \n", table)
	fmt.Printf("Use AddColumn to add more columns to the table. \n")
}

// AddColumn adds a column to the table.
//
// Example: AddColumn(database, "yarak_schema", "kürek_table", "kazancı","bedih")
// Yarak_schema is the schema name, kürek_table is the table name, kazancı is the column name and bedih is the column type.
//
// You can check https://www.w3schools.com/sql/sql_datatypes.asp for available datatypes.
func AddColumn(db *sql.DB, schema string, table string, colname string, coltype string) {
	columnCreateQuery := fmt.Sprintf("alter table %s.%s\n    add %s %s\n", schema, table, colname, coltype)
	_, err := db.Exec(columnCreateQuery)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Done!")
}

///////////////////////////////////////////
// DATA INSERTION && DELETION FUNCTIONS //
//////////////////////////////////////////
func rowCount(db *pgx.Conn, schema string, table string, columnname string) int {
	var rowCount int
	// Check if database is alive.
	err := db.Ping(context.Background())
	if err != nil {
		panic(err)
	}
	if db == nil {
		err = errors.New("CreateEmployee: db is null")
		panic(err)
	}
	// Get the row count.
	rowCountQuery := fmt.Sprintf("SELECT COUNT(*) as %s\nFROM %s.%s", columnname, schema, table)
	err = db.QueryRow(context.Background(), rowCountQuery).Scan(&rowCount)
	if err != nil {
		panic(err)
	}
	return rowCount
}

// AddData adds data to a single column. Pass a columnname and a data array.
//
// Prepare the second data map[string]any parameters like string is whatever the fuck, value is anything.
func AddData(db *pgx.Conn, data any, schema string, table string, columnname string) {
	rowC := rowCount(db, schema, table, columnname)
	valueString := fmt.Sprintf("'%s'", data.(string))
	rowQuery := fmt.Sprintf("INSERT INTO %s.%s (%s)\nVALUES (%s);", schema, table, columnname, valueString)
	_, err := db.Exec(context.Background(), rowQuery)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Added data to row %d! \n", rowC)
	rowC++
}

func UpdateData(db *pgx.Conn, schema string, table string, data string, columnname string, prevcolumn string, prevdata any) {
	sqlQ := fmt.Sprintf("UPDATE %s.%s\nSET %s = '%s'\nWHERE %s = '%s'", schema, table, columnname, data, prevcolumn, prevdata.(string))
	resp, err := db.Exec(context.Background(), sqlQ)
	rowsAff := resp.RowsAffected()
	if err != nil {
		panic(err)
	}
	logFile, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	_, err = fmt.Fprintf(logFile, "%d \n", rowsAff)
	if err != nil {
		return
	}
	err = logFile.Close()
	if err != nil {
		return
	}
}
func UpdateDataFloat(db *pgx.Conn, schema string, table string, data float64, columnname string, prevcolumn string, prevdata any) {
	sqlQ := fmt.Sprintf("UPDATE %s.%s\nSET %s = '%f'\nWHERE %s = '%s'", schema, table, columnname, data, prevcolumn, prevdata.(string))
	resp, err := db.Exec(context.Background(), sqlQ)
	if err != nil {
		panic(err)
	}
	rowsAff := resp.RowsAffected()
	logFile, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	_, err = fmt.Fprintf(logFile, "%d \n", rowsAff)
	if err != nil {
		return
	}
	err = logFile.Close()
	if err != nil {
		return
	}
}

// DeleteData deletes data from the table, pass the column number desired to search, and the value to search for on deleteVar.
// Any matches will be deleted.
//
// Dont forget to pass schema and table.
func DeleteData(db *sql.DB, deleteVar string, schema string, table string, column string) (sql.Result, error) {
	tableQ := fmt.Sprintf("%s.%s", schema, table)
	sqlQuery := goqu.Delete(tableQ).From(tableQ).Where(goqu.C(column).Eq(deleteVar))
	sqlReq, args, err := sqlQuery.ToSQL()
	if err != nil {
		panic(err)
	}
	return db.Exec(sqlReq, args...)
}

// DeleteDataAll deletes entire table without deleting table itself.
func DeleteDataAll(db *pgx.Conn, schema string, table string) (pgconn.CommandTag, error) {
	tableQ := fmt.Sprintf("%s.%s", schema, table)
	sqlQuery := goqu.Delete(tableQ).From(tableQ)
	sqlReq, args, err := sqlQuery.ToSQL()
	if err != nil {
		panic(err)
	}
	return db.Exec(context.Background(), sqlReq, args...)
}

// RetrieveData retrieves data from the database and prints it to the console by looping rows.
// Table input should be something like: ExampleSchema.ExampleTable
func RetrieveData(db *sql.DB, table string) error {
	sqlQuery := goqu.Select("*").From(table)
	sqlString, _, err := sqlQuery.ToSQL()
	if err != nil {
		panic(err)
	}
	rows, err := db.QueryContext(context.Background(), sqlString)
	if err != nil {
		panic(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			panic(err)
		}
	}(rows)
	for rows.Next() {
		var id sql.NullInt64
		var name string
		var location string
		err := rows.Scan(&id, &name, &location)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%d %s %s\n", id, name, location)
	}
	return err
}
