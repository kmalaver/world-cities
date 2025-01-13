package main

import "context"

type Exporter interface {
	Export(ctx context.Context, path string, data []Country) error
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
	States         []State           `json:"states" db:"-"`
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
