package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var includeCountriesFlag = flag.String("countries", "", "Comma separated list of countries to include")
var outputPathFlag = flag.String("output", "world-cities.sql", "Path to output file")

func main() {
	flag.Parse()

	var countries []string

	if *includeCountriesFlag != "" {
		countries = strings.Split(*includeCountriesFlag, ",")
		for i := range countries {
			countries[i] = strings.TrimSpace(countries[i])
			countries[i] = strings.ToUpper(countries[i])
		}
	}

	if len(countries) == 0 {
		// load all countries
		files, err := ioutil.ReadDir("data")
		if err != nil {
			panic(err)
		}

		for _, file := range files {
			if file.IsDir() {
				countries = append(countries, file.Name())
			}
		}
	}

	var sql strings.Builder

	// write the schema
	sql.WriteString(`
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
`)

	// write the data
	for _, country := range countries {
		if country == "" {
			continue
		}
		countryStr, err := convertCountryToSQL(country)
		if err != nil {
			panic(err)
		}
		sql.WriteString(countryStr)
		sql.WriteString("\n\n")
	}

	err := ioutil.WriteFile(*outputPathFlag, []byte(sql.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("SQL file written to %s\n", *outputPathFlag)
}

func convertCountryToSQL(country string) (string, error) {

	// check if the country exists
	_, err := os.Stat(fmt.Sprintf("data/%s", country))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("Country %s does not exist", country)
		}
		return "", err
	}

	// load the country data
	countryBytes, err := ioutil.ReadFile(fmt.Sprintf("data/%s/_country.json", country))
	if err != nil {
		return "", err
	}

	var countryData Country
	err = json.Unmarshal(countryBytes, &countryData)
	if err != nil {
		return "", err
	}

	// iterate over the states
	states, err := ioutil.ReadDir(fmt.Sprintf("data/%s", country))
	if err != nil {
		return "", err
	}

	var statesData []State
	var citiesData []City

	for i := range states {
		state := states[i]
		if state.IsDir() {
			continue
		}

		// if the file is _country.json, skip it
		if state.Name() == "_country.json" {
			continue
		}

		stateData, err := ioutil.ReadFile(fmt.Sprintf("data/%s/%s", country, state.Name()))
		if err != nil {
			return "", err
		}

		var stateObj State
		err = json.Unmarshal(stateData, &stateObj)

		if err != nil {
			return "", err
		}

		stateObj.CountryID = countryData.ID
		for i := range stateObj.Cities {
			stateObj.Cities[i].StateID = stateObj.ID
		}

		statesData = append(statesData, stateObj)
		citiesData = append(citiesData, stateObj.Cities...)
	}

	// generate the SQL
	countrySql := mapToInsert("countries", []map[string]any{
		{
			"id":              countryData.ID,
			"name":            countryData.Name,
			"iso3":            countryData.Iso3,
			"iso2":            countryData.Iso2,
			"numeric_code":    countryData.NumericCode,
			"phone_code":      countryData.PhoneCode,
			"capital":         countryData.Capital,
			"currency":        countryData.Currency,
			"currency_name":   countryData.CurrencyName,
			"currency_symbol": countryData.CurrencySymbol,
			"tld":             countryData.Tld,
			"native":          countryData.Native,
			"region":          countryData.Region,
			"subregion":       countryData.Subregion,
			"translations":    countryData.Translations,
			"latitude":        countryData.Latitude,
			"longitude":       countryData.Longitude,
			"emoji":           countryData.Emoji,
			"emoji_u":         countryData.EmojiU,
		},
	})

	statesMap := make([]map[string]any, len(statesData))
	for i, state := range statesData {
		statesMap[i] = map[string]any{
			"id":         state.ID,
			"country_id": state.CountryID,
			"name":       state.Name,
			"state_code": state.StateCode,
			"latitude":   state.Latitude,
			"longitude":  state.Longitude,
			"type":       state.Type,
		}
	}

	if len(statesMap) == 0 {
		return countrySql, nil
	}

	statesSql := mapToInsert("states", statesMap)

	citiesMap := make([]map[string]any, len(citiesData))
	for i, city := range citiesData {
		citiesMap[i] = map[string]any{
			"id":         city.ID,
			"state_id":   city.StateID,
			"country_id": countryData.ID,
			"name":       city.Name,
			"latitude":   city.Latitude,
			"longitude":  city.Longitude,
		}
	}

	if len(citiesMap) == 0 {
		return strings.Join([]string{countrySql, statesSql}, "\n\n"), nil
	}

	citiesSql := mapToInsert("cities", citiesMap)

	return strings.Join([]string{countrySql, statesSql, citiesSql}, "\n\n"), nil
}

type Country struct {
	ID             int               `json:"id" db:"id"`
	Name           string            `json:"name" db:"name"`
	Iso3           string            `json:"iso3" db:"iso3"`
	Iso2           string            `json:"iso2" db:"iso2"`
	NumericCode    string            `json:"numeric_code" db:"numeric_code"`
	PhoneCode      string            `json:"phonecode" db:"phone_code"`
	Capital        string            `json:"capital" db:"capital"`
	Currency       string            `json:"currency" db:"currency"`
	CurrencyName   string            `json:"currency_name" db:"currency_name"`
	CurrencySymbol string            `json:"currency_symbol" db:"currency_symbol"`
	Tld            string            `json:"tld" db:"tld"`
	Native         string            `json:"native" db:"native"`
	Region         string            `json:"region" db:"region"`
	Subregion      string            `json:"subregion" db:"subregion"`
	Translations   map[string]string `json:"translations" db:"translations"`
	Latitude       float64           `json:"latitude" db:"latitude"`
	Longitude      float64           `json:"longitude" db:"longitude"`
	Emoji          string            `json:"emoji" db:"emoji"`
	EmojiU         string            `json:"emojiU" db:"emoji_u"`
}

type State struct {
	ID        int     `json:"id" db:"id"`
	CountryID int     `json:"country_id" db:"country_id"`
	Name      string  `json:"name" db:"name"`
	StateCode string  `json:"state_code" db:"state_code"`
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
	Type      string  `json:"type" db:"type"`
	Cities    []City  `json:"cities" db:"-"`
}

type City struct {
	ID        int     `json:"id" db:"id"`
	StateID   int     `json:"state_id" db:"state_id"`
	Name      string  `json:"name" db:"name"`
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
}

func mapToInsert(tableName string, data []map[string]any) string {
	cols := make([]string, 0, len(data[0]))

	for k := range data[0] {
		cols = append(cols, k)
	}

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
				sql.WriteString(quote(v))
			case int, int8, int16, int32, int64:
				sql.WriteString(fmt.Sprintf("%d", v))
			case float32, float64:
				sql.WriteString(fmt.Sprintf("%f", v))
			default:
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					panic(err)
				}
				sql.WriteString(fmt.Sprintf("'%s'", jsonBytes))
			}
		}
		sql.WriteString(")")
	}

	sql.WriteString(" ON CONFLICT DO NOTHING;")

	return sql.String()
}

func quote(s string) string {
	s = strings.ReplaceAll(s, `'`, `''`)
	if strings.Contains(s, `\`) {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = ` E'` + s + `'`
	} else {
		s = `'` + s + `'`
	}
	return s

}
