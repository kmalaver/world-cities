package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

var includeCountriesFlag = flag.String("countries", "", "Comma separated list of countries to include")
var outputPathFlag = flag.String("output", "world-cities", "Path to output file")
var format = flag.String("format", "psql", "Output format (psql, json)")

var formats = map[string]Exporter{
	"psql": &PsqlExporter{},
	"json": &JsonExporter{},
}

func main() {
	start := time.Now()

	flag.Parse()

	var countries []string
	if *includeCountriesFlag != "" {
		countries = strings.Split(*includeCountriesFlag, ",")
		for i := range countries {
			countries[i] = strings.TrimSpace(countries[i])
			countries[i] = strings.ToUpper(countries[i])
		}
	}

	exporter, ok := formats[*format]
	if !ok {
		fmt.Printf("Invalid format: %s\n", *format)
		os.Exit(1)
	}

	data, err := parseData(countries)
	if err != nil {
		panic(err)
	}

	dataBytes, err := exporter.Export(context.Background(), data)
	if err != nil {
		panic(err)
	}

	outputPath := *outputPathFlag + "." + exporter.Ext()

	err = os.WriteFile(outputPath, dataBytes, 0644)
	if err != nil {
		panic(err)
	}

	statesLen := 0
	citiesLen := 0
	for _, country := range data {
		statesLen += len(country.States)
		for _, state := range country.States {
			citiesLen += len(state.Cities)
		}
	}

	fmt.Printf("Data exported to %s in %s\n", outputPath, time.Since(start))
	fmt.Printf("Countries: %d\n", len(data))
	fmt.Printf("States: %d\n", statesLen)
	fmt.Printf("Cities: %d\n", citiesLen)
}

func parseData(includedCountries []string) ([]Country, error) {
	files, err := os.ReadDir("data")
	if err != nil {
		return nil, err
	}

	var countries []Country
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		if len(includedCountries) > 0 && !slices.Contains(includedCountries, file.Name()) {
			continue
		}

		countryData, err := os.ReadFile(fmt.Sprintf("data/%s/_country.json", file.Name()))
		if err != nil {
			return nil, err
		}

		var country Country
		err = json.Unmarshal(countryData, &country)
		if err != nil {
			return nil, err
		}

		states, err := os.ReadDir(fmt.Sprintf("data/%s", file.Name()))
		if err != nil {
			return nil, err
		}

		for _, state := range states {
			if state.IsDir() {
				continue
			}
			if state.Name() == "_country.json" {
				continue
			}

			stateData, err := os.ReadFile(fmt.Sprintf("data/%s/%s", file.Name(), state.Name()))
			if err != nil {
				return nil, err
			}

			var stateObj State
			err = json.Unmarshal(stateData, &stateObj)
			if err != nil {
				return nil, err
			}

			stateObj.CountryID = country.ID
			for i := range stateObj.Cities {
				stateObj.Cities[i].StateID = stateObj.ID
			}

			country.States = append(country.States, stateObj)
		}

		// sort states by id
		slices.SortFunc(country.States, func(a State, b State) int {
			return a.ID - b.ID
		})

		countries = append(countries, country)
	}

	// sort countries by id
	slices.SortFunc(countries, func(a Country, b Country) int {
		return a.ID - b.ID
	})

	return countries, nil
}
