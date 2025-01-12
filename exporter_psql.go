package main

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type PsqlExporter struct{}

func (e *PsqlExporter) Ext() string {
	return "sql"
}

func (e *PsqlExporter) Export(ctx context.Context, countries []Country) ([]byte, error) {

	var countriesData []map[string]any
	for i := range countries {
		country := countries[i]
		countriesData = append(countriesData, map[string]any{
			"id":              country.ID,
			"name":            country.Name,
			"iso3":            country.Iso3,
			"iso2":            country.Iso2,
			"numeric_code":    country.NumericCode,
			"phone_code":      country.PhoneCode,
			"capital":         country.Capital,
			"currency":        country.Currency,
			"currency_name":   country.CurrencyName,
			"currency_symbol": country.CurrencySymbol,
			"tld":             country.Tld,
			"native":          country.Native,
			"region":          country.Region,
			"subregion":       country.Subregion,
			"translations":    country.Translations,
			"latitude":        country.Latitude,
			"longitude":       country.Longitude,
			"emoji":           country.Emoji,
			"emoji_u":         country.EmojiU,
		})
	}

	var statesData []map[string]any
	for _, country := range countries {
		for i := range country.States {
			state := country.States[i]
			statesData = append(statesData, map[string]any{
				"id":         state.ID,
				"country_id": state.CountryID,
				"name":       state.Name,
				"state_code": state.StateCode,
				"latitude":   state.Latitude,
				"longitude":  state.Longitude,
				"type":       state.Type,
			})

		}
	}

	var citiesData []map[string]any
	for _, country := range countries {
		for _, state := range country.States {
			for i := range state.Cities {
				city := state.Cities[i]
				citiesData = append(citiesData, map[string]any{
					"id":         city.ID,
					"state_id":   city.StateID,
					"country_id": state.CountryID,
					"name":       city.Name,
					"latitude":   city.Latitude,
					"longitude":  city.Longitude,
				})
			}
		}
	}

	schema := e.generateSchema()
	countriesSql := e.generateInserts("countries", countriesData)
	statesSql := e.generateInserts("states", statesData)
	citiesSql := e.generateInserts("cities", citiesData)

	return []byte(strings.Join([]string{schema, countriesSql, statesSql, citiesSql}, "\n\n")), nil
}

func (e *PsqlExporter) generateInserts(tableName string, data []map[string]any) string {
	const batchSize = 500
	var inserts []string
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}
		inserts = append(inserts, e.mapToInsert(tableName, data[i:end]))
	}

	return strings.Join(inserts, "\n")
}

func (e *PsqlExporter) mapToInsert(tableName string, data []map[string]any) string {
	cols := make([]string, 0, len(data[0]))

	for k := range data[0] {
		cols = append(cols, k)
	}

	// sort columns to ensure consistent order in generated SQL but keep id as first column
	slices.SortFunc(cols, func(a string, b string) int {
		if a == "id" {
			return -1
		}
		if b == "id" {
			return 1
		}
		return strings.Compare(a, b)
	})

	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES\n", tableName, strings.Join(cols, ", ")))

	for i, row := range data {
		if i > 0 {
			sql.WriteString(",\n")
		}
		sql.WriteString("(")
		for j, col := range cols {
			if j > 0 {
				sql.WriteString(", ")
			}
			switch v := row[col].(type) {
			case string:
				sql.WriteString(e.quote(v))
			case int, int8, int16, int32, int64:
				sql.WriteString(fmt.Sprintf("%d", v))
			case float32, float64:
				sql.WriteString(fmt.Sprintf("%f", v))
			default:
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					panic(err)
				}
				sql.WriteString(fmt.Sprintf("%s", e.quote(string(jsonBytes))))
			}
		}
		sql.WriteString(")")
	}

	sql.WriteString("\nON CONFLICT DO UPDATE SET ")
	for i, col := range cols {
		if col == "id" {
			continue
		}
		if i > 1 {
			sql.WriteString(", ")
		}
		sql.WriteString(fmt.Sprintf("%s = EXCLUDED.%s", col, col))
	}
	sql.WriteString(";")

	return sql.String()
}

func (e *PsqlExporter) quote(s string) string {
	s = strings.ReplaceAll(s, `'`, `''`)
	if strings.Contains(s, `\`) {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = ` E'` + s + `'`
	} else {
		s = `'` + s + `'`
	}
	return s

}

func (e *PsqlExporter) generateSchema() string {
	return `
CREATE TABLE IF NOT EXISTS countries (
	id INT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	iso3 VARCHAR(3) NOT NULL,
	iso2 VARCHAR(2) NOT NULL,
	numeric_code VARCHAR(3) NOT NULL,
	phone_code VARCHAR(5) NOT NULL,
	capital VARCHAR(255) NOT NULL,
	currency VARCHAR(255) NOT NULL,
	currency_name VARCHAR(255) NOT NULL,
	currency_symbol VARCHAR(255) NOT NULL,
	tld VARCHAR(255) NOT NULL,
	native VARCHAR(255) NOT NULL,
	region VARCHAR(255) NOT NULL,
	subregion VARCHAR(255) NOT NULL,
	translations JSON NOT NULL,
	latitude double precision NOT NULL,
	longitude double precision NOT NULL,
	emoji VARCHAR(255) NOT NULL,
	emoji_u VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS states (
	id INT PRIMARY KEY,	
	country_id INT NOT NULL REFERENCES countries(id),
	name VARCHAR(255) NOT NULL,
	state_code VARCHAR(255) NOT NULL,
	latitude double precision NOT NULL,
	longitude double precision NOT NULL,
	type VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS cities (
	id bigint PRIMARY KEY,
	country_id INT NOT NULL REFERENCES countries(id),
	state_id INT NOT NULL REFERENCES states(id), 
	name VARCHAR(255) NOT NULL,
	latitude double precision NOT NULL,
	longitude double precision NOT NULL
);
`
}
